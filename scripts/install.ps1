param (
    [Parameter(Mandatory)]
    $observe_collection_endpoint, 
    [Parameter(Mandatory)]
    $observe_token
)

$installer_url="https://github.com/observeinc/observe-agent/releases/download/v0.1.39/observe-agent_Windows_x86_64.zip"
$local_installer="c:\temp\observe-agent_Windows_x86_64.zip"
$observeagent_install_dir="$env:ProgramFiles\Observe Agent"
$temp_dir="c:\temp"

New-Item -ItemType Directory -Force -Path $temp_dir
New-Item -ItemType Directory -Force -Path $otel_install_dir 

Invoke-WebRequest -Uri $installer_url -OutFile $local_installer

Expand-Archive -LiteralPath $local_installer -DestinationPath $temp_dir
Copy-Item -Path $temp_dir\observe-agent_Windows_x86_64\observe-agent.exe -Destination $observeagent_install_dir
Copy-Item -Path $temp_dir\observe-agent_Windows_x86_64\otel-collector.yaml.exe -Destination $observeagent_install_dir

if(-not (Test-Path "${otel_install_dir}\otelcol-contrib.exe")){
    try{
        tar -xzf $local_installer -C $otel_install_dir
    }catch [System.Management.Automation.CommandNotFoundException] {
        Write-Host "tar not found, trying 7z"
        & "$env:ProgramFiles\7-zip\7z" x $local_installer -o"$temp_dir" -aoa
        & "$env:ProgramFiles\7-zip\7z" x $local_installer.Replace(".gz", "") -o"$otel_install_dir" -aoa
    }    
}else{
    Write-Host "Found existing otel installation, skipping installation and moving on to configuration."
}

# Read the content of the config.yaml file
$configContent = Get-Content -Path $otel_install_config -Raw

# Replace ${myhost} with the actual value
$configContent = $configContent -replace '\${OBSERVE_COLLECTION_ENDPOINT}', $observe_collection_endpoint
$configContent = $configContent -replace '\${OBSERVE_TOKEN}', $observe_token

# Write the modified content back to the config.yaml file
$configContent | Set-Content -Path $otel_install_config

# example script that installs observe-agent as a windows service
$params = @{
    Name = "ObserveAgent"
    BinaryPathName = "C:\Users\konstantin\work\observe-agent\agent.exe C:\Users\konstantin\work\observe-agent.yaml"
    DisplayName = "Observe Agent"
    StartupType = "Automatic"
    Description = "Observe Agent based on OpenTelemetry collector"
  }
  
  New-Service @params

if(-not (Get-Service OpenTelemetry -ErrorAction SilentlyContinue)){
    New-Service -Name "OpenTelemetry" -BinaryPathName "`"${otel_install_dir}\otelcol-contrib.exe`" --config `"${otel_install_dir}\config.yaml`""
    Start-Service OpenTelemetry
    }
else{
    Stop-Service OpenTelemetry
    Start-Service OpenTelemetry
}