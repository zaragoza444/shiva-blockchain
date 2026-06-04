# DNS + HTTPS preflight for novatrustee.digital
param(
    [string]$VpsIp = ""
)

$Domain = "novatrustee.digital"
$ParkingIps = @('107.161.23.204', '198.251.81.30', '209.141.38.71')
Write-Host "=== OneX preflight: $Domain ===" -ForegroundColor Cyan

try {
    $ips = @(Resolve-DnsName $Domain -Type A -ErrorAction Stop | ForEach-Object { $_.IPAddress } | Sort-Object -Unique)
    Write-Host "DNS A: $($ips -join ', ')" -ForegroundColor Green
    $parking = $ips | Where-Object { $ParkingIps -contains $_ }
    if ($parking.Count -gt 0 -and $ips.Count -ge $parking.Count) {
        Write-Host "BLOCKED: Domain still on registrar parking IPs. Fix DNS first:" -ForegroundColor Red
        Write-Host "  1. Log in at your .digital registrar"
        Write-Host "  2. DNS -> delete extra A records for novatrustee"
        Write-Host "  3. Add ONE A record -> your VPS public IPv4"
        if ($VpsIp) { Write-Host "  4. That IP should be: $VpsIp" -ForegroundColor Yellow }
    }
    if ($ips.Count -gt 1) {
        Write-Host "WARN: Multiple A records - use only your VPS IP." -ForegroundColor Yellow
    }
    if ($VpsIp -and $ips -notcontains $VpsIp) {
        Write-Host "WARN: DNS does not include your VPS IP $VpsIp yet." -ForegroundColor Yellow
    }
    if ($VpsIp -and $ips -contains $VpsIp -and $ips.Count -eq 1) {
        Write-Host "DNS OK for VPS $VpsIp" -ForegroundColor Green
    }
} catch {
    Write-Host "DNS: not resolving - $($_.Exception.Message)" -ForegroundColor Red
}

foreach ($url in @(
    "https://$Domain/health",
    "https://$Domain/bridge/platform/status",
    "https://$Domain/wallet/"
)) {
    try {
        $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 12
        Write-Host "OK $($r.StatusCode) $url" -ForegroundColor Green
    } catch {
        Write-Host "FAIL $url - $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "After DNS points to your VPS, on Ubuntu run:" -ForegroundColor Cyan
Write-Host '  CERTBOT_EMAIL=you@digital ./scripts/deploy-vps-novatrustee.sh'
Write-Host ""
Write-Host "Check DNS with your VPS IP:" -ForegroundColor Cyan
Write-Host '  .\scripts\deploy-vps-novatrustee.ps1 -VpsIp 203.0.113.10'
