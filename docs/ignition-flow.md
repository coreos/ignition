```mermaid
flowchart TB
    %% ===== IGNITION BOOT FLOW =====

    %% --- GRUB Firstboot Detection ---
    boot["Boot"] --> grub["GRUB bootloader"]

    subgraph GRUB_FIRSTBOOT ["GRUB Firstboot Detection"]
        direction TB
        grub_check{"/ignition.firstboot
        stamp file on bootfs?"}
        grub_check -->|Yes| grub_source["Source /ignition.firstboot
        (may set ignition_network_kcmdline)"]
        grub_source --> grub_append["Append to kernel cmdline:
        ignition.firstboot $ignition_network_kcmdline"]
        grub_check -->|No| grub_no_flag["No ignition.firstboot on cmdline"]
    end
    grub --> GRUB_FIRSTBOOT

    %% --- Initramfs Generator ---
    GRUB_FIRSTBOOT --> generator["ignition-generator
    (reads /proc/cmdline)"]
    generator --> firstboot_check{"ignition.firstboot
    on kernel cmdline?"}

    %% --- Subsequent Boot Path ---
    firstboot_check -->|No| subsequent_target["ignition-subsequent.target"]
    subsequent_target --> subsequent_diskful["ignition-diskful-subsequent.target"]
    subsequent_diskful --> subsequent_done["Ignition services do not run.
    Boot continues normally."]

    %% --- Firstboot Path ---
    firstboot_check -->|Yes| complete_gate["ignition-complete.target activated"]
    complete_gate --> fetch_offline["ignition-fetch-offline.service"]

    %% --- Fetch Offline Details ---
    subgraph FETCH_OFFLINE ["Ignition Fetch Offline"]
        direction TB
        offline_check_cmdline{"Config provided
        via kernel cmdline?"}
        offline_check_user_ign{"/usr/lib/ignition/user.ign
        exists?"}
        offline_try_platform["Try platform provider"]
        offline_write_cache["Write config to /run/ignition.json"]
        offline_needs_net{"Config needs
        network resources?"}
        offline_signal_neednet["Signal neednet"]
        offline_done["Done"]
        offline_check_cmdline -->|Yes| offline_write_cache
        offline_check_cmdline -->|No| offline_check_user_ign
        offline_check_user_ign -->|Yes| offline_write_cache
        offline_check_user_ign -->|No| offline_try_platform
        offline_try_platform -->|Config found| offline_write_cache
        offline_try_platform -->|Needs network| offline_signal_neednet
        offline_write_cache --> offline_needs_net
        offline_needs_net -->|Yes| offline_signal_neednet
        offline_needs_net -->|No| offline_done
    end
    fetch_offline --> FETCH_OFFLINE
    
    FETCH_OFFLINE --> fetch_check{"/run/ignition.json exists?"}
    fetch_check -->|Yes, skip ignition-fetch.service| kargs_service
    fetch_check -->|No| fetch_service["ignition-fetch.service"]
    
    %% --- Fetch Service Details ---
    subgraph FETCH_ONLINE ["Ignition Fetch"]
        direction TB
        online_check_cmdline{"Config provided
        via kernel cmdline?"}
        online_check_user_ign{"/usr/lib/ignition/user.ign
        exists?"}
        online_fetch_provider["Fetch from platform provider
        (see Provider Specific Behavior - Config Fetch below)"]
        online_write_config["Write config to /run/ignition.json"]
        online_done["Done"]
        online_check_cmdline -->|Yes| online_write_config
        online_check_cmdline -->|No| online_check_user_ign
        online_check_user_ign -->|Yes| online_write_config
        online_check_user_ign -->|No| online_fetch_provider
        online_fetch_provider -->|Config found| online_write_config
        online_fetch_provider -->|No config| online_done
        online_write_config --> online_done
    end
    fetch_service --> FETCH_ONLINE
    
    %% --- Disk & Mount Services ---
    FETCH_ONLINE --> kargs_service["ignition-kargs.service"]
    kargs_service -->|kargs changed| reboot_kargs["Reboot & restart from top"]
    
    kargs_service -->|no changes| disks_service["ignition-disks.service"]
    disks_service --> diskful_target["ignition-diskful.target reached"]
    diskful_target --> mount_service["ignition-mount.service"]
    
    %% --- Files ---
    mount_service --> files_service["ignition-files.service"]
    initrd_root_fs_target["initrd-root-fs.target"] --> afterburn_hostname_service["afterburn-hostname.service"]
    afterburn_hostname_service -.-> files_service
    
    %% --- Files Service Details ---
    subgraph FILES ["Ignition Files"]
        direction TB
        files_read_cache["Read cached config
        from /run/ignition.json"]
        files_apply["Merge with base configs and apply
        (create users, write files,
        directories, links)"]
        files_done["Done"]
        files_read_cache --> files_apply
        files_apply --> files_done
    end
    files_service --> FILES
    
    FILES --> complete_target["ignition-complete.target reached"]
    
    complete_target --> delete_config["ignition-delete-config.service"]
    
    %% ===== STYLING =====
    classDef service fill:#42a5f5,stroke:#1565c0,stroke-width:2px,color:#000
    classDef target fill:#ffa726,stroke:#e65100,stroke-width:2px,color:#000
    
    class fetch_offline,fetch_service,kargs_service,disks_service,mount_service,files_service,afterburn_hostname_service,delete_config service
    class diskful_target,complete_target,complete_gate,network_target,initrd_root_fs_target,subsequent_target,subsequent_diskful target

```

## Provider Specific Behavior
### Config Fetch
#### Azure
```mermaid
flowchart TB
    %% ===== AZURE PROVIDER-SPECIFIC CONFIG FETCH =====

    start["Fetch provider-specific config"] --> imds_request["HTTP GET to Azure IMDS
    http://169.254.169.254/metadata/instance/compute/userData
    ?api-version=2021-01-01&format=text
    Header - Metadata: true"]

    imds_request --> imds_retry{"Response code?"}
    imds_retry -->|"404, 410, 429, or 5xx
    Retry with exponential backoff
    (200ms initial, 5s max)"| imds_request
    imds_retry -->|"Network Unreachable
    (DHCP has not completed)"| imds_request
    imds_retry -->|200, empty body| fallback_ovf
    imds_retry -->|200, has body| write_config["Write decoded config to /run/ignition.json"]
    imds_retry -->|Other error| error["Error"]
    write_config --> done["Done"]

    fallback_ovf["Fallback: read OVF custom data from CD-ROM device"]
    fallback_ovf --> scan["Scan for UDF CD-ROM (often /dev/sr0)"]
    scan --> mount["Mount device"]
    mount --> read["Read for ovf-env.xml and CustomData.bin"]
    read --> available{"Config available?"}
    available -->|Yes| write_device["Write config to /run/ignition.json"]
    write_device --> done
    available -->|No| wait["Wait 1s"] --> scan
```