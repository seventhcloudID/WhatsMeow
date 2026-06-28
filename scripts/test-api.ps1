$Base = "http://localhost:8081"
$Key = "your-secret-api-key"
$H = @{"X-API-Key" = $Key}

function Test-EP($name, $method, $url, $headers=$H, $body=$null, $expectSuccess=$true) {
    try {
        $params = @{Uri=$url; Method=$method; Headers=$headers; ErrorAction="Stop"}
        if ($body) { $params["Body"] = $body; $params["ContentType"] = "application/json" }
        $r = Invoke-RestMethod @params
        $ok = $r.success -eq $expectSuccess
        Write-Host ("[$(if($ok){'PASS'}else{'FAIL'})] $name") -ForegroundColor $(if($ok){"Green"}else{"Red"})
        if (-not $ok) { Write-Host "  Response: $($r | ConvertTo-Json -Compress)" }
        return $r
    } catch {
        $fail = -not $expectSuccess
        Write-Host ("[$(if($fail){'PASS'}else{'FAIL'})] $name (HTTP error)") -ForegroundColor $(if($fail){"Green"}else{"Red"})
        Write-Host "  $($_.Exception.Message)"
        return $null
    }
}

Write-Host "`n=== WhatsApp Gateway API Test ===`n" -ForegroundColor Cyan

Test-EP "Health (no auth)" GET "$Base/health" @{} 
Test-EP "Auth reject" GET "$Base/session/status" @{"X-API-Key"="wrong"} $null $false
Test-EP "Session status" GET "$Base/session/status"
Test-EP "Session connect" POST "$Base/session/connect"
Start-Sleep -Seconds 2
$qr = Test-EP "Get QR" GET "$Base/session/qr"
if ($qr.data.code) { Write-Host "  QR code tersedia (timeout: $($qr.data.timeout_seconds)s)" -ForegroundColor Gray }
Test-EP "Connect (idempotent)" POST "$Base/session/connect"
Test-EP "Get webhook" GET "$Base/webhook"
Test-EP "Set webhook" PUT "$Base/webhook" $H '{"url":"http://localhost:9999/webhook"}'
Test-EP "Logout (belum login)" POST "$Base/session/logout" $H $null $false
Test-EP "Send text (belum login)" POST "$Base/message/text" $H '{"to":"6281234567890","text":"test"}' $false
Test-EP "Send text (invalid body)" POST "$Base/message/text" $H '{"to":"","text":""}' $false
Test-EP "Pair (invalid phone)" POST "$Base/session/pair" $H '{"phone":"123"}' $false
Test-EP "Disconnect" POST "$Base/session/disconnect"
Start-Sleep -Seconds 1
Test-EP "Reconnect after disconnect" POST "$Base/session/connect"
Start-Sleep -Seconds 2
Test-EP "QR after reconnect" GET "$Base/session/qr"
Test-EP "Clear webhook" PUT "$Base/webhook" $H '{"url":""}'

Write-Host "`n=== Selesai ===`n" -ForegroundColor Cyan
