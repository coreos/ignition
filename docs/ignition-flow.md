```mermaid
flowchart TB
    %% ===== IGNITION BOOT FLOW =====

    %% --- Early Boot ---
    boot["Boot"] --> setup_pre["ignition-setup-pre.service"]
    setup_pre --> setup["ignition-setup.service"]
    setup --> fetch_offline["ignition-fetch-offline.service"]

    %% --- Fetch Offline Details ---
    subgraph FETCH_OFFLINE ["Ignition Fetch Offline"]
        direction TB
        offline_detect_platform["Detect platform"]
        offline_check_configs["Check configs at:"]
        offline_base_dir["/usr/lib/ignition/base.d"]
        offline_platform_dir["/usr/lib/ignition/base.platform.d/{platform}"]
        offline_detect_platform --> offline_check_configs
        offline_check_configs --> offline_base_dir
        offline_check_configs --> offline_platform_dir
        offline_check_user_ign{"/usr/lib/ignition/user.ign exists?"}
        offline_base_dir --> offline_check_user_ign
        offline_platform_dir --> offline_check_user_ign
        offline_check_user_ign -->|Yes| offline_copy_user_ign["Write to /run/ignition.json"]
        offline_check_user_ign -->|No| offline_done["Done"]
        offline_copy_user_ign --> offline_done
    end
    fetch_offline --> FETCH_OFFLINE
    
    FETCH_OFFLINE --> fetch_check{"/run/ignition.json exists?"}
    fetch_check -->|Yes, skip ignition-fetch.service| kargs_service
    fetch_check -->|No| fetch_service["ignition-fetch.service"]
    
    %% --- Fetch Service Details ---
    subgraph FETCH_ONLINE ["Ignition Fetch"]
        direction TB
        online_detect_platform["Detect platform"]
        online_check_configs["Check configs at:"]
        online_base_dir["/usr/lib/ignition/base.d"]
        online_platform_dir["/usr/lib/ignition/base.platform.d/{platform}"]
        online_detect_platform --> online_check_configs
        online_check_configs --> online_base_dir
        online_check_configs --> online_platform_dir
        online_request_cloud_configs["Request cloud specific configs"]
        online_cloud_configs_present{"Cloud configs present?"}
        online_open_config_device["Open and read config device"]
        online_check_user_ign{"/usr/lib/ignition/user.ign exists?"}
        online_base_dir --> online_check_user_ign
        online_platform_dir --> online_check_user_ign
        online_check_user_ign -->|Yes| online_copy_user_ign["Write config to /run/ignition.json"]
        online_check_user_ign -->|No| online_request_cloud_configs
        online_copy_user_ign --> online_done["Done"]
        online_request_cloud_configs --> online_cloud_configs_present
        online_cloud_configs_present -->|Yes| online_write_cloud["Write config to /run/ignition.json"]
        online_write_cloud --> online_done
        online_cloud_configs_present -->|No| online_open_config_device
        online_config_device_present{"Config present?"}
        online_open_config_device --> online_config_device_present
        online_config_device_present -->|Yes| online_write_device["Write configto /run/ignition.json"]
        online_write_device --> online_done
        online_config_device_present -->|No| online_done
    end
    fetch_service --> FETCH_ONLINE
    
    %% --- Network Stack ---
    subgraph NETWORK ["Network Stack"]
        direction TB
        networkd_service["systemd-networkd.service"]
        network_config["systemd-networkd.service - Network Configuration"]
        network_target["network.target reached"]
        networkd_service -->  network_config --> network_target
    end
    setup --> NETWORK
    NETWORK --> FETCH_ONLINE
    NETWORK --> get_dhcp_address["Get DHCP address"]
    get_dhcp_address --> online_request_cloud_configs
    
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
        files_detect_platform["Detect platform"]
        files_check_configs["Check configs at:"]
        files_base_dir["/usr/lib/ignition/base.d"]
        files_platform_dir["/usr/lib/ignition/base.platform.d/{platform}"]
        files_detect_platform --> files_check_configs
        files_check_configs --> files_base_dir
        files_check_configs --> files_platform_dir
        files_base_dir --> files_check_run_json
        files_platform_dir --> files_check_run_json
        files_check_run_json{"/run/ignition.json exists?"}
        files_check_run_json -->|Yes| files_merge_all["Merge all detected configs and apply"]
        files_merge_all --> files_done["Done"]
        files_check_run_json -->|No| files_error["Error"]
    end
    files_service --> FILES
    
    FILES --> quench_service["ignition-quench.service"]
    quench_service --> initrd_setup_root["initrd-setup-root-after-ignition.service"]
    quench_service --> complete_target["ignition-complete.target"]
    
    %% ===== STYLING =====
    classDef service fill:#42a5f5,stroke:#1565c0,stroke-width:2px,color:#000
    classDef target fill:#ffa726,stroke:#e65100,stroke-width:2px,color:#000
    
    class setup_pre,setup,fetch_offline,fetch_service,kargs_service,disks_service,mount_service,files_service,quench_service,initrd_setup_root,network_config,networkd_service,afterburn_hostname_service service
    class diskful_target,complete_target,network_target,initrd_root_fs_target target

```