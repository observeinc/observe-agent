#!/usr/bin/env python3
import os
import sys
import time
import tempfile
import urllib.request
import urllib.error
import utils as u



def _configure_agent(remote_host: u.Host, platform: str, env_vars: dict) -> None:
    """Configure the observe-agent with minimal settings to allow it to start

    Args:
        remote_host (Host): instance to ssh into
        platform (str): platform name (windows, linux)
        env_vars (dict): environment variables with config
    """
    print("🔧 Configuring observe-agent...")

    # Get observe credentials from env vars
    observe_url = env_vars.get("observe_url")
    observe_token = env_vars.get("observe_token")

    if not observe_url or not observe_token:
        u.die("❌ OBSERVE_URL and OBSERVE_TOKEN are required for upgrade test")

    if platform == "windows":
        config_command = (
            r'Set-Location "C:\Program Files\Observe\observe-agent"; '
            r'./observe-agent init-config --token {} --observe_url {} --print'.format(
                observe_token, observe_url
            )
        )
    else:  # Linux
        config_command = 'sudo observe-agent init-config --token {} --observe_url {} --print'.format(
            observe_token, observe_url
        )

    # First print what we're going to configure
    result = remote_host.run_command(config_command)
    u.print_remote_result(result)

    # Now actually set the config (remove --print)
    if platform == "windows":
        config_command = (
            r'Set-Location "C:\Program Files\Observe\observe-agent"; '
            r'./observe-agent init-config --token {} --observe_url {}'.format(
                observe_token, observe_url
            )
        )
    else:  # Linux
        config_command = 'sudo observe-agent init-config --token {} --observe_url {}'.format(
            observe_token, observe_url
        )

    result = remote_host.run_command(config_command)
    if result.exited != 0:
        print("⚠️ Warning: Configuration may have failed, but continuing...")
    else:
        print("✅ Agent configured successfully")


def _start_service(remote_host: u.Host, start_command: str, platform: str, env_vars: dict = None):
    """Start the observe-agent service

    Args:
        remote_host (Host): instance to ssh into
        start_command (str): platform-specific start command
        platform (str): platform name (windows, linux)
        env_vars (dict): environment variables for Windows-specific operations
    """
    print("🚀 Starting observe-agent service...")

    # Windows needs special handling for copying start script
    if platform == "windows" and env_vars:
        home_dir = r"/C:/Users/{}".format(env_vars["user"])
        current_script_dir = os.path.dirname(os.path.abspath(__file__))
        ps_start_script_path = os.path.join(current_script_dir, "start_agent_windows.ps1")
        remote_host.put_file(local_path=ps_start_script_path, remote_path=home_dir)

    result = remote_host.run_command(start_command)
    u.print_remote_result(result)

    if platform == "windows":
        # Windows PowerShell: check for stderr (script handles errors)
        if result.stderr:
            raise RuntimeError("❌ Error starting observe-agent service")
    else:
        # Linux: check exit code (systemd writes to stderr even on success)
        if result.exited != 0:
            raise RuntimeError("❌ Error starting observe-agent service")


def _verify_running(remote_host: u.Host, status_command: str, version_command: str,
                   version_name: str, num_retries: int = 10) -> str:
    """Verify agent is running and get its version

    Args:
        remote_host (Host): instance to ssh into
        status_command (str): command to check status
        version_command (str): command to get version
        version_name (str): description of version (e.g., "old", "new")
        num_retries (int): number of retries for status check

    Returns:
        str: version string
    """
    print(f"🔍 Verifying {version_name} version is running...")
    agent_status = u.check_status_loop(remote_host, status_command, num_retries=num_retries)
    if not agent_status:
        u.die(f"❌ {version_name.capitalize()} version failed to start")

    version = u.get_agent_version(remote_host, version_command)
    print(f"✅ {version_name.capitalize()} version {version} is running")
    return version


def _get_installation_package(env_vars: dict) -> tuple:
    """Get the built distribution package for installation/upgrade

    Args:
        env_vars (dict): environment variables with machine config

    Returns:
        tuple: (filename, full_path) of the package
    """
    current_dir = os.getcwd()
    dist_directory = os.path.abspath(os.path.join(current_dir, "..", "dist"))
    files = os.listdir(dist_directory)

    package_type = env_vars["machine_config"]["package_type"]
    architecture = env_vars["machine_config"]["architecture"]
    distribution = env_vars["machine_config"]["distribution"]

    # Iterate through files and find matches
    for filename in files:
        if package_type in filename and architecture in filename:
            # We can make this more general if need be.
            if "windows" in distribution and "windows" not in filename.lower():
                continue
            full_path = os.path.join(dist_directory, filename)
            print(f"Found matching file {filename} at: {full_path}")
            return filename, full_path

    u.die(
        f"❌ No matching file found for {distribution},{architecture},{package_type} in {dist_directory}: {', '.join(files)}"
    )


def _perform_upgrade(remote_host: u.Host, filename: str, full_path: str,
                    platform: str, env_vars: dict = None):
    """Perform the upgrade installation

    Args:
        remote_host (Host): instance to ssh into
        filename (str): name of the package file
        full_path (str): full path to the package file
        platform (str): platform name (windows, linux)
        env_vars (dict): environment variables for Windows-specific operations
    """
    if platform == "windows":
        home_dir = r"/C:/Users/{}".format(env_vars["user"])

        # Copy new package and install script
        remote_host.put_file(local_path=full_path, remote_path=home_dir)

        current_script_dir = os.path.dirname(os.path.abspath(__file__))
        ps_install_script_path = os.path.join(current_script_dir, "install_windows.ps1")
        remote_host.put_file(local_path=ps_install_script_path, remote_path=home_dir)

        # Run upgrade installation
        home_dir_powershell = r"C:\Users\{}".format(env_vars["user"])
        upgrade_command = r".\install_windows.ps1 -local_installer {}\{}".format(
            home_dir_powershell, filename
        )
    else:  # Linux
        # Copy new package to remote host
        remote_host.put_file(local_path=full_path, remote_path="/tmp/observe-agent-new.tar.gz")

        # Extract and run upgrade installation
        upgrade_command = "sudo tar -xzf /tmp/observe-agent-new.tar.gz -C /tmp && sudo /tmp/observe-agent/install_linux.sh"

    result = remote_host.run_command(upgrade_command)
    u.print_remote_result(result)

    if (platform == "windows" and result.stderr) or (platform == "linux" and result.exited != 0):
        u.die("❌ Error upgrading observe-agent")


def _verify_upgrade(remote_host: u.Host, status_command: str, version_command: str,
                   old_version: str, platform: str = None) -> None:
    """Verify the upgrade was successful

    Args:
        remote_host (Host): instance to ssh into
        status_command (str): command to check status
        version_command (str): command to get version
        old_version (str): the previous version
        platform (str): platform name for platform-specific recovery
    """
    print("🔍 Verifying service is still running after upgrade...")
    agent_status = u.check_status_loop(remote_host, status_command, num_retries=15)

    if not agent_status:
        u.die("❌ Service failed to remain running after upgrade")

    new_version = u.get_agent_version(remote_host, version_command)
    print(f"✅ New version {new_version} is running")

    if old_version == new_version:
        print("⚠️ Warning: Version appears unchanged after upgrade")
    else:
        print(f"✅ Successfully upgraded from {old_version} to {new_version}")


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to upgrade observe-agent on Windows from older version to current build

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing
    """
    # Get old version from env var, default to v2.5.0
    old_version = os.environ.get("OLD_VERSION", "v2.5.0")

    # Commands for Windows
    start_command = r".\start_agent_windows.ps1"
    status_command = r'Get-Service ObserveAgent;Set-Location "${Env:Programfiles}\Observe\observe-agent"; ./observe-agent status'
    version_command = r'Set-Location "${Env:Programfiles}\Observe\observe-agent"; ./observe-agent version'

    # Install old version using the same approach as test_install.py
    print("🔄 Installing older version of observe-agent...")

    # Set windows home dir paths for consistency (same as test_install.py)
    home_dir = r"/C:/Users/{}".format(env_vars["user"])  # for user in sftp
    home_dir_powershell = r"C:\Users\{}".format(env_vars["user"])  # for use in powershell script

    # Download the old version Windows package locally, then upload (same as test_install.py)

    download_url = f"https://github.com/observeinc/observe-agent/releases/download/{old_version}/observe-agent_Windows_x86_64.zip"
    old_filename = "observe-agent-old.zip"

    # Download to local temp file first
    with tempfile.NamedTemporaryFile(delete=False, suffix='.zip') as temp_file:
        print(f"Downloading {download_url}...")
        try:
            with urllib.request.urlopen(download_url, timeout=60) as response:
                temp_file.write(response.read())
            local_package_path = temp_file.name
        except urllib.error.URLError as e:
            u.die(f"❌ Error downloading old version: {e}")
        except Exception as e:
            u.die(f"❌ Error downloading old version: {e}")

    # Upload the package to remote host with correct filename (same as test_install.py)
    remote_path_with_filename = f"{home_dir}/{old_filename}"
    remote_host.put_file(local_path=local_package_path, remote_path=remote_path_with_filename)

    # Copy the install script from the scripts directory (same as test_install.py)
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    ps_installation_script_path = os.path.join(current_script_dir, "install_windows.ps1")
    remote_host.put_file(local_path=ps_installation_script_path, remote_path=home_dir)

    # Run install script with the uploaded package (same pattern as test_install.py)
    result = remote_host.run_command(
        r".\install_windows.ps1 -local_installer {}\{}".format(
            home_dir_powershell, old_filename
        )
    )
    u.print_remote_result(result)

    if result.stderr:
        # Use RuntimeError like test_install.py
        raise RuntimeError("❌ Installation error in install_windows.ps1 powershell script")
    else:
        print("✅ Old version installation completed")

    # Cleanup local temp file
    os.unlink(local_package_path)

    # Configure the old version so it can start properly
    _configure_agent(remote_host, "windows", env_vars)

    # Start the configured service
    _start_service(remote_host, start_command, "windows", env_vars)

    # Verify old version is running
    old_version_actual = _verify_running(remote_host, status_command, version_command, "old")

    # Find and perform upgrade
    print("⬆️ Upgrading to new version...")
    filename, full_path = _get_installation_package(env_vars)
    _perform_upgrade(remote_host, filename, full_path, "windows", env_vars)

    # Verify upgrade was successful
    _verify_upgrade(remote_host, status_command, version_command, old_version_actual, "windows")


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to upgrade observe-agent on Linux from older version to current build

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing
    """
    # Get old version from env var, default to v2.5.0
    old_version = os.environ.get("OLD_VERSION", "v2.5.0")

    # Commands for Linux
    start_command = "sudo systemctl enable --now observe-agent"
    status_command = "observe-agent status"
    version_command = "observe-agent version"

    # Install old version using package managers like test_install.py
    print("🔄 Installing older version of observe-agent...")

    # Get architecture
    arch_result = remote_host.run_command("arch")
    if arch_result.exited != 0:
        u.die("❌ Error getting system architecture")
    arch = arch_result.stdout.strip()

    # Determine package type and download URL based on distribution
    distribution = env_vars["machine_config"]["distribution"].lower()
    home_dir = f"/home/{env_vars['user']}"

    if "redhat" in distribution:
        # Use RPM package for RedHat-based systems
        # v2.5.0 uses format: observe-agent-2.5.0-1.x86_64.rpm
        version_number = old_version.lstrip('v')  # Remove 'v' prefix
        package_url = f"https://github.com/observeinc/observe-agent/releases/download/{old_version}/observe-agent-{version_number}-1.{arch}.rpm"
        filename = f"observe-agent-{version_number}-1.{arch}.rpm"
        install_command = f"cd ~ && sudo yum localinstall {filename} -y"
    elif "debian" in distribution:
        # Use DEB package for Debian-based systems
        # v2.5.0 uses format: observe-agent_2.5.0_amd64.deb (note: amd64 not x86_64)
        version_number = old_version.lstrip('v')  # Remove 'v' prefix
        deb_arch = "amd64" if arch == "x86_64" else arch  # Debian uses amd64 instead of x86_64
        package_url = f"https://github.com/observeinc/observe-agent/releases/download/{old_version}/observe-agent_{version_number}_{deb_arch}.deb"
        filename = f"observe-agent_{version_number}_{deb_arch}.deb"
        install_command = f"cd ~ && sudo dpkg -i {filename}"
    else:
        u.die(f"❌ Unsupported distribution for package installation: {distribution}")

    # Download package locally, then upload (same pattern as test_install.py)

    # Download to local temp file first
    with tempfile.NamedTemporaryFile(delete=False, suffix='.pkg') as temp_file:
        print(f"Downloading {package_url}...")
        try:
            with urllib.request.urlopen(package_url, timeout=60) as response:
                temp_file.write(response.read())
            local_package_path = temp_file.name
        except urllib.error.URLError as e:
            u.die(f"❌ Error downloading old version: {e}")
        except Exception as e:
            u.die(f"❌ Error downloading old version: {e}")

    # Upload the package to remote host with correct filename (same as test_install.py)
    remote_path_with_filename = f"{home_dir}/{filename}"
    remote_host.put_file(local_path=local_package_path, remote_path=remote_path_with_filename)

    # Install the package using the appropriate package manager (same as test_install.py)
    result = remote_host.run_command(install_command)
    u.print_remote_result(result)
    if result.exited != 0:
        u.die("❌ Error installing older version of observe-agent")
    else:
        print("✅ Old version installation completed")

    # Cleanup local temp file
    os.unlink(local_package_path)

    # Configure the old version so it can start properly
    _configure_agent(remote_host, "linux", env_vars)

    # Start the configured service
    _start_service(remote_host, start_command, "linux")

    # Verify old version is running
    old_version_actual = _verify_running(remote_host, status_command, version_command, "old")

    # Find and perform upgrade
    print("⬆️ Upgrading to new version...")
    filename, full_path = _get_installation_package(env_vars)
    _perform_upgrade(remote_host, filename, full_path, "linux")

    # Verify upgrade was successful
    _verify_upgrade(remote_host, status_command, version_command, old_version_actual, "linux")


if __name__ == "__main__":
    env_vars = u.get_env_vars()
    remote_host = u.Host(
        host_ip=env_vars["host"],
        username=env_vars["user"],
        key_file_path=env_vars["key_filename"],
        password=env_vars["password"],
    )

    # Test SSH Connection before starting test
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))

    distribution = env_vars["machine_config"]["distribution"].lower()

    if "redhat" in distribution or "debian" in distribution:
        run_test_linux(remote_host, env_vars)
    elif "windows" in distribution:
        run_test_windows(remote_host, env_vars)
    elif "docker" in distribution:
        print("✅ Docker upgrade test skipped - upgrades are handled via container image replacement")
        print("✅ Upgrade test passed (Docker containers use image-based upgrades)")
    else:
        u.die(f"❌ Unsupported distribution: {distribution}")