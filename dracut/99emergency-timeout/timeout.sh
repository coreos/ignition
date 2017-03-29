# Before starting the emergency shell, prompt the user to press Enter.
# If they don't, reboot the system.
#
# Assumes /bin/sh is bash.

_prompt_for_timeout() {
    local timeout=300
    local interval=15

    if [[ -e /.emergency-shell-confirmed ]]; then
        return
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
            echo "Press Enter for emergency shell or wait $m $m_label $s seconds for reboot."
        else
            echo "Press Enter for emergency shell or wait $m $m_label for reboot."
        fi

        local anything
        if read -t $interval anything; then
            > /.emergency-shell-confirmed
            return
        fi
        timeout=$(( $timeout - $interval ))
    done

    echo "Rebooting."
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
