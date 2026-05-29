package installtest

import "path/filepath"

type termuxDryRunFixture struct {
	Root        string
	Home        string
	InstallHome string
	Prefix      string
}

func newTermuxDryRunFixture(root string) termuxDryRunFixture {
	return termuxDryRunFixture{
		Root:        root,
		Home:        filepath.Join(root, "home"),
		InstallHome: filepath.Join(root, "install-home"),
		Prefix:      filepath.Join(root, "com.termux", "files", "usr"),
	}
}

func (f termuxDryRunFixture) env(extra map[string]string) map[string]string {
	env := map[string]string{
		"HOME":                        f.Home,
		"GORMES_INSTALL_HOME":         f.InstallHome,
		"GORMES_SKIP_SETUP":           "1",
		"GORMES_RESTART_GATEWAY":      "never",
		"GORMES_INSTALL_TEST_UNAME_M": "aarch64",
		"PREFIX":                      f.Prefix,
		"TERMUX_VERSION":              "0.119.0",
	}
	for k, v := range extra {
		env[k] = v
	}
	return env
}
