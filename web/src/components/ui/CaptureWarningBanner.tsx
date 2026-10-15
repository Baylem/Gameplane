import { Alert } from "@heroui/react";
import { TriangleAlert } from "lucide-react";
import { useOptionalIntl } from "@/lib/i18n/format";

interface CaptureWarningBannerProps {
  retentionHours?: number;
  onDismiss?: () => void;
}

export function CaptureWarningBanner({ retentionHours = 24, onDismiss }: CaptureWarningBannerProps) {
  const intl = useOptionalIntl();
  return (
    <Alert status="warning" className="flex items-start gap-3">
      <Alert.Indicator className="mt-0.5 shrink-0">
        <TriangleAlert className="h-5 w-5 text-warning-soft-foreground" />
      </Alert.Indicator>
      <Alert.Content className="flex flex-1 flex-col gap-2">
        <Alert.Title className="text-sm font-medium">{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.caution_network_packet_captures_contain", defaultMessage: "Caution: network packet captures contain real player data", description: "Capture warning banner: text in Alert.Title" })}</Alert.Title>
        <ul className="space-y-1 text-xs text-muted">
          <li>{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.player_ip_addresses_and_port_numbers", defaultMessage: "• Player IP addresses and port numbers", description: "Capture warning banner: text in li" })}</li>
          <li>{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.network_timing_and_game_protocol", defaultMessage: "• Network timing and game protocol messages", description: "Capture warning banner: text in li" })}</li>
          <li>{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.for_some_games_in_band_credentials", defaultMessage: "• For some games, in-band credentials (passwords, tokens, session keys)", description: "Capture warning banner: text in li" })}</li>
        </ul>
        <Alert.Description className="text-xs">{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.captures_are_not_redacted_or_sanitized", defaultMessage: "Captures are not redacted or sanitized. Access is restricted to administrators only. Captures are automatically deleted after the configured retention window (currently {retentionHours} {retentionHours, plural, one {hour} other {hours}}).", description: "Capture warning banner: paragraph in Alert.Description; {retentionHours} is the capture retention window in hours" }, { retentionHours: retentionHours })}</Alert.Description>
        {onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            className="mt-2 inline-flex text-xs font-medium text-warning-soft-foreground hover:underline"
          >{intl.formatMessage({ id: "web.components.ui.capture_warning_banner.dismiss", defaultMessage: "Dismiss", description: "Capture warning banner: text in button" })}</button>
        )}
      </Alert.Content>
    </Alert>
  );
}
