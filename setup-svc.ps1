# example script that installs observe-agent as a windows service
$params = @{
  Name = "ObserveAgent"
  BinaryPathName = "C:\Users\konstantin\work\observe-agent\agent.exe C:\Users\konstantin\work\observe-agent.yaml"
  DisplayName = "Observe Agent"
  StartupType = "Automatic"
  Description = "Observe Agent based on OpenTelemetry collector"
}

New-Service @params

