# Before starting the emergency shell, prompt the user to press Enter.
# If they don't, reboot the system.
#
# Assumes /bin/sh is bash.

# _wait_for_journalctl_to_stop will block until either:
# - no messages have appeared in journalctl for the past 5 seconds
# - 15 seconds have elapsed
_wait_for_journalctl_to_stop() {
    local time_since_last_log=0

    local time_started="$(date '+%s')"
    local now="$(date '+%s')"

    while [ ${time_since_last_log} -lt 5 -a $((now-time_started)) -lt 15 ]; do
        sleep 1

        local last_log_timestamp="$(journalctl -e -n 1 -q -o short-unix | cut -d '.' -f 1)"
        local now="$(date '+%s')"

        local time_since_last_log=$((now-last_log_timestamp))
    done
}

_prompt_for_timeout() {
    local timeout=300
    local interval=15

    if [[ -e /.emergency-shell-confirmed ]]; then
        return
    fi
    failed=$(systemctl --failed --no-legend | cut -f 1 -d ' ')
    if [ -n "${failed}" ]; then
        # Something failed, suppress kernel logs so that it's more likely
        # the useful bits from the journal are available.
        dmesg --console-off

        # There's a couple straggler systemd messages. Wait until it's been 5
        # seconds since something was written to the journal.
        _wait_for_journalctl_to_stop

        # Print Ignition logs
        if echo ${failed} | grep -qFe 'ignition-'; then
        cat <<EOF
------
Ignition has failed. Please ensure your config is valid. Note that only
Ignition spec v3.0.0+ configs are accepted.

A CLI validation tool to check this called ignition-validate can be
downloaded from GitHub:
    https://github.com/coreos/ignition/releases
------

EOF
        fi
        echo "Displaying logs from failed units: ${failed}"
        for unit in ${failed}; do
            # 10 lines should be enough for everyone
            journalctl -b --no-pager --no-hostname -u ${unit} -n 10
        done
    fi

    # Regularly prompt with time remaining.  This ensures the prompt doesn't
    # get lost among kernel and systemd messages, and makes it clear what's
    # going on if the user just connected a serial console.
    while [[ $timeout > 0 ]]; do
        local m=$(( $timeout / 60 ))
        local s=$(( $timeout % 60 ))
        local m_label="minutes"
        if [[ $m = 1 ]]; then
            m_label="minute"
        fi

        if [[ $s != 0 ]]; then
            echo -n -e "Press Enter for emergency shell or wait $m $m_label $s seconds for reboot.      \r"
        else
            echo -n -e "Press Enter for emergency shell or wait $m $m_label for reboot.                 \r"
        fi

        local anything
        if read -t $interval anything; then
            > /.emergency-shell-confirmed
            return
        fi
        timeout=$(( $timeout - $interval ))
    done

    echo -e "\nRebooting."
    # This is not very nice, but since reboot.target likely conflicts with
    # the existing goal target wrt the desired state of shutdown.target,
    # there doesn't seem to be a better option.
    systemctl reboot --force
    exit 0
}

# If we're invoked from a dracut breakpoint rather than
# dracut-emergency.service, we won't have a controlling terminal and stdio
# won't be connected to it. Explicitly read/write /dev/console.
_prompt_for_timeout < /dev/console > /dev/console
