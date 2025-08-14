package agentresourceextension

type Config struct {
	LocalFilePath string `mapstructure:"local_file_path"`
}

func (cfg *Config) Validate() error {
	return nil
}