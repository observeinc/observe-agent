param (
    [Parameter()]
    [String]$version,
    [Parameter()]
    [String]$zip_dir,
    [Parameter()]
    [String]$observe_collection_endpoint,
    [Parameter()]
    [String]$observe_token
)

$installer_url="https://github.com/observeinc/observe-agent/releases/latest/download/observe-agent_Windows_x86_64.zip"
if ($PSBoundParameters.ContainsKey('version')){
    if ($version -match '^\d') {
        $version="v$version"
    }
    $installer_url="https://github.com/observeinc/observe-agent/releases/download/$version/observe-agent_Windows_x86_64.zip"
}
$local_installer="C:\temp\observe-agent_Windows_x86_64.zip"
$program_data_filestorage="C:\ProgramData\Observe\observe-agent\filestorage"
$observeagent_install_dir="$env:ProgramFiles\Observe\observe-agent"
$temp_dir="C:\temp"

New-Item -ItemType Directory -Force -Path $temp_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir\config
New-Item -ItemType Directory -Force -Path $observeagent_install_dir\connections
New-Item -ItemType Directory -Force -Path $program_data_filestorage

if ($PSBoundParameters.ContainsKey('zip_dir')){
    Write-Output "Installing from provided zip file: $zip_dir..."
    $local_installer=$zip_dir
} else {
    Write-Output "Downloading observe-agent from $installer_url..."
    Invoke-WebRequest -Uri $installer_url -OutFile $local_installer
}

# Stop the observe agent if its running so that we can copy the new .exe
if((Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    Stop-Service ObserveAgent
}

Expand-Archive -Force -LiteralPath $local_installer -DestinationPath "$temp_dir\observe-agent_Windows_x86_64"
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\observe-agent.exe -Destination $observeagent_install_dir
if (Test-Path $temp_dir\observe-agent_Windows_x86_64\otel-collector.yaml) {
    Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\otel-collector.yaml -Destination $observeagent_install_dir\config\otel-collector.yaml
}
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\connections\* -Destination $observeagent_install_dir\connections -Recurse

# If there's already an observe-agent.yaml we don't copy the template from the downloaded installation package and leave existing observe-agent.yaml alone
if (-Not (Test-Path "$observeagent_install_dir\observe-agent.yaml"))
{
    Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\observe-agent.yaml -Destination $observeagent_install_dir

    # We can only insert param templates if there's no existing config and we're copying over the template observe-agent.yaml
    $configContent = Get-Content -Path $observeagent_install_dir\observe-agent.yaml -Raw

    # Replace $OBSERVE_COLLECTION_ENDPOINT and $OBSERVE_TOKEN with param if provided
    if($PSBoundParameters.ContainsKey('observe_collection_endpoint'))
    {
        $configContent = $configContent -replace '\${OBSERVE_COLLECTION_ENDPOINT}', $observe_collection_endpoint
    }
    if($PSBoundParameters.ContainsKey('observe_token'))
    {
        $configContent = $configContent -replace '\${OBSERVE_TOKEN}', $observe_token
    }

    # Write the modified content back to the config.yaml file
    $configContent | Set-Content -Path $observeagent_install_dir\observe-agent.yaml
}

if(-not (Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    $params = @{
        Name = "ObserveAgent"
        BinaryPathName =  "`"${observeagent_install_dir}\observe-agent.exe`" `"${observeagent_install_dir}\observe-agent.yaml`""
        DisplayName = "Observe Agent"
        StartupType = "Automatic"
        Description = "Observe Agent based on OpenTelemetry collector"
      }

    New-Service @params
    Start-Service ObserveAgent
    }
else{
    Stop-Service ObserveAgent
    Start-Service ObserveAgent
}