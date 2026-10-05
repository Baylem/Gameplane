package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword(context.Background(), "correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword(context.Background(), "correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("verify correct password: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(context.Background(), "wrong guess", hash)
	if err != nil {
		t.Fatalf("verify wrong: err=%v", err)
	}
	if ok {
		t.Fatal("wrong password accepted")
	}
}

func TestHashPerCallDiffers(t *testing.T) {
	SetFastHashParams(t)
	// Salt randomness means two hashes of the same password differ.
	a, _ := HashPassword(context.Background(), "x")
	b, _ := HashPassword(context.Background(), "x")
	if a == b {
		t.Fatal("hashes must differ across calls due to salt")
	}
}

func TestHashPassword_CancelledContextReturnsWhileSlotsHeld(t *testing.T) {
	SetFastHashParams(t)
	if err := argonSem.Acquire(context.Background(), 2); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer argonSem.Release(2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		_, err := HashPassword(ctx, "password")
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("HashPassword did not return promptly on a cancelled context")
	}
}

func TestVerifyPassword_CancelledContextReturnsWhileSlotsHeld(t *testing.T) {
	SetFastHashParams(t)
	if err := argonSem.Acquire(context.Background(), 2); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer argonSem.Release(2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		_, err := VerifyPassword(ctx, "password", "x")
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("VerifyPassword did not return promptly on a cancelled context")
	}
}
