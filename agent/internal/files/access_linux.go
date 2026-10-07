package files

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"

	"golang.org/x/sys/unix"
)

// Linux's POSIX ACL xattr is a little-endian version followed by 8-byte
// entries. A named-user entry lets the unprivileged agent preserve the old
// owner's access without CAP_CHOWN or making a private file group-readable.
const (
	accessACLName        = "system.posix_acl_access"
	aclUserObj    uint16 = 1
	aclUser       uint16 = 2
	aclGroupObj   uint16 = 4
	aclGroup      uint16 = 8
	aclMask       uint16 = 16
	aclOther      uint16 = 32
	aclUndefined  uint32 = 0xffffffff
)

var errPreserveAccess = errors.New("cannot preserve existing file access")

type aclEntry struct {
	tag, perm uint16
	id        uint32
}

type fileAccess struct {
	exists bool
	stat   unix.Stat_t
	acl    []byte
}

func decodeACL(raw []byte) ([]aclEntry, error) {
	if len(raw) < 4 || (len(raw)-4)%8 != 0 || binary.LittleEndian.Uint32(raw) != 2 {
		return nil, errors.New("invalid POSIX ACL encoding")
	}
	var entries []aclEntry
	for off := 4; off < len(raw); off += 8 {
		e := aclEntry{binary.LittleEndian.Uint16(raw[off:]), binary.LittleEndian.Uint16(raw[off+2:]), binary.LittleEndian.Uint32(raw[off+4:])}
		if e.perm > 7 {
			return nil, errors.New("invalid POSIX ACL permissions")
		}
		switch e.tag {
		case aclUserObj, aclUser, aclGroupObj, aclGroup, aclMask, aclOther:
		default:
			return nil, errors.New("invalid POSIX ACL entry")
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func encodeACL(entries []aclEntry) []byte {
	raw := make([]byte, 4+8*len(entries))
	binary.LittleEndian.PutUint32(raw, 2)
	for i, e := range entries {
		off := 4 + i*8
		binary.LittleEndian.PutUint16(raw[off:], e.tag)
		binary.LittleEndian.PutUint16(raw[off+2:], e.perm)
		binary.LittleEndian.PutUint32(raw[off+4:], e.id)
	}
	return raw
}

func replacementACL(st *unix.Stat_t, raw []byte, newUID uint32) ([]byte, error) {
	if st.Uid == newUID {
		return raw, nil
	}
	entries := []aclEntry{{aclUserObj, uint16(st.Mode >> 6 & 7), aclUndefined}, {aclGroupObj, uint16(st.Mode >> 3 & 7), aclUndefined}, {aclOther, uint16(st.Mode & 7), aclUndefined}}
	oldMask := uint16(7)
	if len(raw) > 0 {
		var err error
		entries, err = decodeACL(raw)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.tag == aclMask {
				oldMask = e.perm
			}
		}
	}
	ownerPerm := uint16(st.Mode >> 6 & 7)
	mask := ownerPerm
	out := make([]aclEntry, 0, len(entries)+2)
	for _, e := range entries {
		if e.tag == aclMask || (e.tag == aclUser && e.id == st.Uid) {
			continue
		}
		if e.tag == aclUser || e.tag == aclGroup || e.tag == aclGroupObj {
			// Freeze each old entry's effective permissions before widening
			// the mask; otherwise formerly masked grants would become live.
			e.perm &= oldMask
			mask |= e.perm
		}
		out = append(out, e)
	}
	out = append(out, aclEntry{aclUser, ownerPerm, st.Uid}, aclEntry{aclMask, mask, aclUndefined})
	sort.Slice(out, func(i, j int) bool {
		if out[i].tag != out[j].tag {
			return out[i].tag < out[j].tag
		}
		return out[i].id < out[j].id
	})
	return encodeACL(out), nil
}

func (h *handler) destAccess(dirFD int, parents []string, name string) (fileAccess, error) {
	h.fire("stat", parents, name)
	fd, err := openat(dirFD, name, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return fileAccess{}, nil
	}
	if err != nil {
		return fileAccess{}, err
	}
	defer closeFD(fd)
	a := fileAccess{exists: true}
	if err := unix.Fstat(fd, &a.stat); err != nil {
		return a, err
	}
	if a.stat.Mode&unix.S_IFMT == unix.S_IFLNK {
		return a, h.symlinkError(dirFD, parents, name)
	}
	if a.stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return a, fmt.Errorf("%w: target is not a regular file", errPreserveAccess)
	}
	// O_PATH anchors the inode without needing content-read permission on a
	// private game-owned file. Linux cannot fgetxattr an O_PATH descriptor;
	// this procfs descriptor link addresses that same pinned inode instead.
	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	n, err := unix.Getxattr(path, accessACLName, nil)
	if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.EOPNOTSUPP) {
		return a, nil
	}
	if err != nil {
		return a, fmt.Errorf("%w: read ACL: %w", errPreserveAccess, err)
	}
	a.acl = make([]byte, n)
	n, err = unix.Getxattr(path, accessACLName, a.acl)
	if err != nil {
		return a, fmt.Errorf("%w: read ACL: %w", errPreserveAccess, err)
	}
	a.acl = a.acl[:n]
	return a, nil
}

func (a fileAccess) apply(f *os.File) error {
	mode := os.FileMode(0o644)
	if a.exists {
		mode = fileModeFromStat(a.stat.Mode).Perm()
		var current unix.Stat_t
		if err := unix.Fstat(int(f.Fd()), &current); err != nil {
			return err
		}
		if current.Gid != a.stat.Gid {
			// Keeping the group is necessary to retain ACL_GROUP_OBJ semantics.
			// An unprivileged process can select its own supplementary groups.
			if err := f.Chown(-1, int(a.stat.Gid)); err != nil {
				return err
			}
		}
		raw, err := replacementACL(&a.stat, a.acl, current.Uid)
		if err != nil {
			return err
		}
		if err := f.Chmod(mode); err != nil {
			return err
		}
		if len(raw) > 0 {
			return unix.Fsetxattr(int(f.Fd()), accessACLName, raw, 0)
		}
	}
	// Remove any access ACL inherited from the destination directory when
	// the original had none. chmod alone preserves inherited named grants.
	if err := unix.Fremovexattr(int(f.Fd()), accessACLName); err != nil && !errors.Is(err, unix.ENODATA) && !errors.Is(err, unix.EOPNOTSUPP) {
		return err
	}
	return f.Chmod(mode)
}

func (a fileAccess) unchanged(dirFD int, name string) bool {
	var st unix.Stat_t
	err := unix.Fstatat(dirFD, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return !a.exists
	}
	if err != nil {
		return false
	}
	// Replacing a raced-in symlink is safe: rename does not follow it.
	if st.Mode&unix.S_IFMT == unix.S_IFLNK {
		return true
	}
	return a.exists && st.Dev == a.stat.Dev && st.Ino == a.stat.Ino && st.Ctim == a.stat.Ctim && st.Uid == a.stat.Uid && st.Gid == a.stat.Gid && st.Mode == a.stat.Mode
}
