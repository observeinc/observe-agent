# This scripts take an $local_install parameter, unzips the observe-agent .zip file and
# and copies the relevant files to C:\Program Files\Observe\observe-agent
# It's intended to only install observe-agent on a windows machine and ensure no issues take place

#$local_installer="C:\Users\Administrator\observe-agent_Windows_x86_64.zip" This is set from python 

param (
    [Parameter(Mandatory=$true)]
    [string]$local_installer
)
#Parameter is local_installer .zip file (which should already exist on machine)
Write-Output "Local installer path is located at: $local_installer"

$program_data_filestorage="C:\ProgramData\Observe\observe-agent\filestorage"
$observeagent_install_dir="$env:ProgramFiles\Observe\observe-agent"
$temp_dir="C:\temp"

#Create directories for temp & observe-agent installation ls
New-Item -ItemType Directory -Force -Path $temp_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir\config
New-Item -ItemType Directory -Force -Path $program_data_filestorage

# Stop the observe agent if its running so that we can copy the new .exe
if((Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    Write-Output "Observe Agent is running, Stopping Observe Agent..."
    Stop-Service ObserveAgent
}

# Unzip the installer .zip to C:\temp\observe-agent_extract
# Eg: Unzip C:\Users\Administrator\observe-agent_Windows_x86_64.zip to C:\temp\observe-agent_extract
Write-Output "Unzipping installer $local_installer to $temp_dir\observe-agent_extract"
Expand-Archive -Force -LiteralPath $local_installer  -DestinationPath "$temp_dir\observe-agent_extract"

# Copy relevant files from C:\temp\observe-agent_extract to C:\Program Files\Observe\observe-agent
Write-Output "Copying files from $temp_dir\observe-agent_extract to $observeagent_install_dir"
Copy-Item -Force -Path $temp_dir\observe-agent_extract\observe-agent.exe -Destination $observeagent_install_dir
Copy-Item -Force -Path $temp_dir\observe-agent_extract\observe-agent.yaml -Destination $observeagent_install_dir
Copy-Item -Force -Path $temp_dir\observe-agent_extract\otel-collector.yaml -Destination $observeagent_install_dir\config\otel-collector.yaml
Copy-Item -Force -Path $temp_dir\observe-agent_extract\connections\ -Destination $observeagent_install_dir\connections -Recurse


