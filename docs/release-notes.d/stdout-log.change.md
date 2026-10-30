Modify `--log-to-stdout` to log to stdout along with syslog, if available.
Pair it with `StandardOutput=kmsg` (see `99ignition-log-kmsg`) to forward Ignition logs to kmsg while retaining them in the journal.
`--log-to-stdout` can duplicate journal entries both when stdout is the journal and with that kmsg pairing, where `journalctl -t ignition` sees syslog, stdout, and kernel (journald reading back the kmsg copy).
