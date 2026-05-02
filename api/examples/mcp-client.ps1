param(
    [string]$Config = "api/examples/mcp-client.config.example.json",
    [switch]$ListToolsOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Read-JsonFile {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Config file not found: $Path"
    }

    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json -Depth 100
}

function Build-McpUri {
    param($ConfigObject)

    if ([string]::IsNullOrWhiteSpace($ConfigObject.endpoint)) {
        throw "Config field 'endpoint' is required."
    }

    $builder = [System.UriBuilder]::new($ConfigObject.endpoint)
    $query = [System.Web.HttpUtility]::ParseQueryString($builder.Query)
    if (-not [string]::IsNullOrWhiteSpace($ConfigObject.subject_key)) {
        $query["subject"] = $ConfigObject.subject_key
    }
    $builder.Query = $query.ToString()
    return $builder.Uri
}

function Send-JsonRpcMessage {
    param(
        [System.Net.WebSockets.ClientWebSocket]$Socket,
        [hashtable]$Payload
    )

    $json = $Payload | ConvertTo-Json -Depth 100 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    $segment = [System.ArraySegment[byte]]::new($bytes)
    $Socket.SendAsync(
        $segment,
        [System.Net.WebSockets.WebSocketMessageType]::Text,
        $true,
        [System.Threading.CancellationToken]::None
    ).GetAwaiter().GetResult()
}

function Receive-JsonRpcMessage {
    param([System.Net.WebSockets.ClientWebSocket]$Socket)

    $buffer = New-Object byte[] 4096
    $stream = New-Object System.IO.MemoryStream

    while ($true) {
        $segment = [System.ArraySegment[byte]]::new($buffer)
        $result = $Socket.ReceiveAsync(
            $segment,
            [System.Threading.CancellationToken]::None
        ).GetAwaiter().GetResult()

        if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
            return $null
        }

        if ($result.Count -gt 0) {
            $stream.Write($buffer, 0, $result.Count)
        }

        if ($result.EndOfMessage) {
            break
        }
    }

    $json = [System.Text.Encoding]::UTF8.GetString($stream.ToArray())
    if ([string]::IsNullOrWhiteSpace($json)) {
        return $null
    }
    return $json | ConvertFrom-Json -Depth 100
}

function Write-Section {
    param(
        [string]$Title,
        $Value
    )

    Write-Host ""
    Write-Host "=== $Title ==="
    $Value | ConvertTo-Json -Depth 100
}

Add-Type -AssemblyName System.Web

$configPath = Resolve-Path -LiteralPath $Config
$configObject = Read-JsonFile -Path $configPath

if ([string]::IsNullOrWhiteSpace($configObject.access_token)) {
    throw "Config field 'access_token' is required."
}

$uri = Build-McpUri -ConfigObject $configObject
$socket = [System.Net.WebSockets.ClientWebSocket]::new()
$socket.Options.SetRequestHeader("Authorization", "Bearer $($configObject.access_token)")

try {
    Write-Host "Connecting to $uri"
    $socket.ConnectAsync($uri, [System.Threading.CancellationToken]::None).GetAwaiter().GetResult()

    Send-JsonRpcMessage -Socket $socket -Payload @{
        jsonrpc = "2.0"
        id = 1
        method = "initialize"
        params = @{
            protocolVersion = "2024-11-05"
            capabilities = @{}
            clientInfo = @{
                name = "brights-powershell-demo"
                version = "1.0.0"
            }
        }
    }
    $initializeResponse = Receive-JsonRpcMessage -Socket $socket
    Write-Section -Title "initialize" -Value $initializeResponse

    Send-JsonRpcMessage -Socket $socket -Payload @{
        jsonrpc = "2.0"
        method = "notifications/initialized"
    }

    Send-JsonRpcMessage -Socket $socket -Payload @{
        jsonrpc = "2.0"
        id = 2
        method = "tools/list"
    }
    $toolsResponse = Receive-JsonRpcMessage -Socket $socket
    Write-Section -Title "tools/list" -Value $toolsResponse

    if (-not $ListToolsOnly -and -not [string]::IsNullOrWhiteSpace($configObject.tool_name)) {
        $arguments = @{}
        if ($null -ne $configObject.arguments) {
            $arguments = @{}
            foreach ($property in $configObject.arguments.PSObject.Properties) {
                $arguments[$property.Name] = $property.Value
            }
        }

        Send-JsonRpcMessage -Socket $socket -Payload @{
            jsonrpc = "2.0"
            id = 3
            method = "tools/call"
            params = @{
                name = $configObject.tool_name
                arguments = $arguments
            }
        }
        $toolResponse = Receive-JsonRpcMessage -Socket $socket
        Write-Section -Title "tools/call" -Value $toolResponse
    }

    $socket.CloseAsync(
        [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
        "done",
        [System.Threading.CancellationToken]::None
    ).GetAwaiter().GetResult()
}
finally {
    $socket.Dispose()
}
