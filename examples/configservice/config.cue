package config

#Config: {
	preferences: #Preferences
	backend:     #BackendConfig
	logger:      #LoggerConfig
}

#ThemeMode:  "auto" | "light" | "dark"
#AccentMode: "auto" | "custom"
#Language:   "en" | "pl"

#Preferences: {
	theme:         #ThemeMode | *"auto"    @go(Theme)
	accent_mode:   #AccentMode | *"custom" @go(AccentMode)
	accent_color:  string | *"#F0B429"     @go(AccentColor)
	language:      #Language | *"en"       @go(Language)
	close_to_tray: bool | *true            @go(CloseToTray)
}

#BackendConfig: {
	base_url: string & =~"^https?://" | *"http://127.0.0.1:8080" @go(BaseURL)
}

#LoggerConfig: {
	verbose:      bool | *false                      @go(Verbose)
	no_color:     bool | *false                      @go(NoColor)
	enable_files: bool | *false                      @go(EnableFiles)
	dir:          string | *"./logs"                 @go(Dir)
	prefix:       string | *""                       @go(Prefix)
	level:        string | *"INFO"                   @go(Level)
	max_size_mb:  int & >0 | *10                     @go(MaxSizeMb)
	max_backups:  int & >=0 | *5                     @go(MaxBackups)
	max_age_days: int & >0 | *30                     @go(MaxAgeDays)
	log_name:     string | *"app.log"           @go(LogName)
	compression:  "none" | "gzip" | "zstd" | *"zstd" @go(Compression)
	local_time:   bool | *true                       @go(LocalTime)
}
