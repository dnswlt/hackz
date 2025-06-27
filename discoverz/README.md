# Discoverz

Detect machines on your local network via UDP Multicast.

## Windows Firewall

If your Windows hosts cannot be found, it might be that Windows Firewall
blocks Multicast on port 12345 (the default port used by `discoverz`).

You can add a firewall rule to open this port in PowerShell:

```powershell
New-NetFirewallRule -DisplayName "Allow Multicast UDP 12345" `
   -Direction Inbound `
   -Protocol UDP `
   -LocalPort 12345 `
   -Action Allow `
   -Profile Any
```

To remove the rule later:

```powershell
Remove-NetFirewallRule -DisplayName "Allow Multicast UDP 12345"
```
