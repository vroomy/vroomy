// Package vroomy builds HTTP/HTTPS services from registered Go plugins and TOML
// configuration. Register plugin instances before calling New or NewWithConfig,
// then call ListenUntilSignal to run the configured listeners and close the service
// when listening ends.
//
// Plugins share a process-wide registry. Construction changes the process working
// directory, calls every plugin's Init, then injects dependencies and calls Load in
// dependency order. Handlers write responses through httpserve.Context.
package vroomy
