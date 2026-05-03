param(
    [string]$Config = ".\api\examples\mcp-client.config.json",
    [switch]$ListToolsOnly
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $Config)) {
    throw "Config file not found: $Config"
}

$cfg = Get-Content -LiteralPath $Config -Raw | ConvertFrom-Json

if ([string]::IsNullOrWhiteSpace($cfg.server_url)) {
    throw "server_url is required"
}
if ([string]::IsNullOrWhiteSpace($cfg.access_token)) {
    throw "access_token is required"
}
if ([string]::IsNullOrWhiteSpace($cfg.subject_key)) {
    throw "subject_key is required"
}

$uriBuilder = [System.UriBuilder]::new($cfg.server_url)
$queryPairs = @()
if (-not [string]::IsNullOrWhiteSpace($uriBuilder.Query)) {
    $queryPairs += $uriBuilder.Query.TrimStart("?")
}
$queryPairs += "subject=$([System.Uri]::EscapeDataString([string]$cfg.subject_key))"
$uriBuilder.Query = ($queryPairs -join "&")
$wsUrl = $uriBuilder.Uri.AbsoluteUri

$socket = [System.Net.WebSockets.ClientWebSocket]::new()
$socket.Options.SetRequestHeader("Authorization", "Bearer $($cfg.access_token)")

try {
    Write-Host "Connecting to $wsUrl"
    $socket.ConnectAsync($uriBuilder.Uri, [Threading.CancellationToken]::None).GetAwaiter().GetResult()

    function Send-JsonMessage {
        param([object]$Payload)
        $json = $Payload | ConvertTo-Json -Depth 20 -Compress
        $bytes = [Text.Encoding]::UTF8.GetBytes($json)
        $segment = [ArraySegment[byte]]::new($bytes)
        $socket.SendAsync(
            $segment,
            [System.Net.WebSockets.WebSocketMessageType]::Text,
            $true,
            [Threading.CancellationToken]::None
        ).GetAwaiter().GetResult()
        Write-Host ">> $json"
    }

    function Receive-JsonMessage {
        $buffer = New-Object byte[] 65536
        $ms = New-Object System.IO.MemoryStream
        try {
            do {
                $segment = [ArraySegment[byte]]::new($buffer)
                $result = $socket.ReceiveAsync($segment, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
                if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
                    throw "Server closed websocket: $($result.CloseStatus) $($result.CloseStatusDescription)"
                }
                $ms.Write($buffer, 0, $result.Count)
            } while (-not $result.EndOfMessage)

            $json = [Text.Encoding]::UTF8.GetString($ms.ToArray())
            Write-Host "<< $json"
            return $json | ConvertFrom-Json -Depth 30
        }
        finally {
            $ms.Dispose()
        }
    }

    Send-JsonMessage @{
        jsonrpc = "2.0"
        id = 1
        method = "initialize"
        params = @{
            protocolVersion = "2024-11-05"
            capabilities = @{}
            clientInfo = @{
                name = "brights-powershell-client"
                version = "1.0.0"
            }
        }
    }
    [void](Receive-JsonMessage)

    Send-JsonMessage @{
        jsonrpc = "2.0"
        method = "notifications/initialized"
    }

    Send-JsonMessage @{
        jsonrpc = "2.0"
        id = 2
        method = "tools/list"
    }
    $toolsResponse = Receive-JsonMessage

    if ($ListToolsOnly) {
        return
    }

    $toolName = [string]$cfg.tool_name
    if ([string]::IsNullOrWhiteSpace($toolName)) {
        throw "tool_name is required"
    }

    $arguments = @{}
    if ($null -ne $cfg.arguments) {
        foreach ($property in $cfg.arguments.PSObject.Properties) {
            $arguments[$property.Name] = $property.Value
        }
    }
    if (-not $arguments.ContainsKey("subject_key")) {
        $arguments["subject_key"] = [string]$cfg.subject_key
    }

    Send-JsonMessage @{
        jsonrpc = "2.0"
        id = 3
        method = "tools/call"
        params = @{
            name = $toolName
            arguments = $arguments
        }
    }
    [void](Receive-JsonMessage)
}
finally {
    if ($null -ne $socket) {
        if ($socket.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
            $socket.CloseAsync(
                [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
                "done",
                [Threading.CancellationToken]::None
            ).GetAwaiter().GetResult()
        }
        $socket.Dispose()
    }
}
