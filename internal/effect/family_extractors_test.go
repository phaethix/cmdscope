package effect_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitEffects(t *testing.T) {
	cases := []struct {
		name      string
		command   string
		cwd       string
		wantKinds []ir.EffectKind
		wantRaws  []string
		wantFlags []ir.Flag
		wantUnk   []ir.UnknownCode
	}{
		{
			name:      "push network",
			command:   "git push origin main",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectNetwork},
			wantRaws:  []string{"git-remote"},
		},
		{
			name:      "push force destructive",
			command:   "git push --force origin main",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectNetwork},
			wantRaws:  []string{"git-remote"},
			wantFlags: []ir.Flag{ir.FlagDestructive},
		},
		{
			name:      "push force with lease destructive",
			command:   "git push --force-with-lease",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectNetwork},
			wantRaws:  []string{"git-remote"},
			wantFlags: []ir.Flag{ir.FlagDestructive},
		},
		{
			name:      "reset hard destructive unknown",
			command:   "git reset --hard",
			cwd:       "/ws",
			wantKinds: nil,
			wantFlags: []ir.Flag{ir.FlagDestructive},
			wantUnk:   []ir.UnknownCode{ir.UnknownEffectsRuntimeDependent},
		},
		{
			name:      "clean destructive unknown",
			command:   "git clean -fd",
			cwd:       "/ws",
			wantKinds: nil,
			wantFlags: []ir.Flag{ir.FlagDestructive},
			wantUnk:   []ir.UnknownCode{ir.UnknownEffectsRuntimeDependent},
		},
		{
			name:      "rm deletes",
			command:   "git rm file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectDelete},
			wantRaws:  []string{"file.txt"},
		},
		{
			name:      "checkout write",
			command:   "git checkout -- file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"file.txt"},
		},
		{
			name:      "restore write",
			command:   "git restore file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"file.txt"},
		},
		{
			name:      "add writes dotgit",
			command:   "git add .",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"./.git"},
		},
		{
			name:      "commit writes dotgit",
			command:   "git commit -m msg",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"./.git"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns, flags := effect.ExtractGit(cmd, 0, cond, tc.cwd)
			require.Len(t, effects, len(tc.wantKinds))
			for i, ef := range effects {
				assert.Equal(t, tc.wantKinds[i], ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
			}
			for _, code := range tc.wantUnk {
				assert.True(t, hasUnknownCode(unknowns, code), "want unknown %s in %+v", code, unknowns)
			}
			for _, fl := range tc.wantFlags {
				assert.Contains(t, flags, fl, "want flag %s in %+v", fl, flags)
			}
		})
	}
}

func hasUnknownCode(unks []ir.Unknown, code ir.UnknownCode) bool {
	for _, u := range unks {
		if u.Code == code {
			return true
		}
	}
	return false
}

func TestMiscEffects(t *testing.T) {
	cases := []struct {
		name      string
		command   string
		cwd       string
		wantKinds []ir.EffectKind
		wantRaws  []string
		wantCert  []ir.Certainty
	}{
		{
			name:      "tee write",
			command:   "tee out.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"out.txt"},
			wantCert:  []ir.Certainty{ir.Certain},
		},
		{
			name:      "tee append conditional",
			command:   "tee -a log.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"log.txt"},
			wantCert:  []ir.Certainty{ir.Conditional},
		},
		{
			name:      "touch write",
			command:   "touch a.txt b.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite, ir.EffectWrite},
			wantRaws:  []string{"a.txt", "b.txt"},
			wantCert:  []ir.Certainty{ir.Certain, ir.Certain},
		},
		{
			name:      "truncate write",
			command:   "truncate -s 0 file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"file.txt"},
			wantCert:  []ir.Certainty{ir.Certain},
		},
		{
			name:      "ln write link read source",
			command:   "ln -s src link",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead, ir.EffectWrite},
			wantRaws:  []string{"src", "link"},
			wantCert:  []ir.Certainty{ir.Certain, ir.Certain},
		},
		{
			name:      "rmdir delete",
			command:   "rmdir olddir",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectDelete},
			wantRaws:  []string{"olddir"},
			wantCert:  []ir.Certainty{ir.Certain},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractMisc(cmd, 0, cond, tc.cwd)
			require.Empty(t, unknowns)
			require.Len(t, effects, len(tc.wantKinds))
			for i, ef := range effects {
				assert.Equal(t, tc.wantKinds[i], ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
				assert.Equal(t, tc.wantCert[i], ef.Certainty)
			}
		})
	}
}

func TestSedEffects(t *testing.T) {
	cases := []struct {
		name      string
		command   string
		cwd       string
		wantKinds []ir.EffectKind
		wantRaws  []string
	}{
		{
			name:      "no -i read only",
			command:   "sed 's/a/b/' file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead},
			wantRaws:  []string{"file.txt"},
		},
		{
			name:      "in-place read+write",
			command:   "sed -i 's/a/b/' file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead, ir.EffectWrite},
			wantRaws:  []string{"file.txt", "file.txt"},
		},
		{
			name:      "in-place with suffix",
			command:   "sed -i.bak 's/a/b/' file.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead, ir.EffectWrite},
			wantRaws:  []string{"file.txt", "file.txt"},
		},
		{
			name:      "script skips first positional",
			command:   "sed -i 's/a/b/' a.txt b.txt",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead, ir.EffectWrite, ir.EffectRead, ir.EffectWrite},
			wantRaws:  []string{"a.txt", "a.txt", "b.txt", "b.txt"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractSed(cmd, 0, cond, tc.cwd)
			require.Empty(t, unknowns)
			require.Len(t, effects, len(tc.wantKinds))
			for i, ef := range effects {
				assert.Equal(t, tc.wantKinds[i], ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
			}
		})
	}
}

func TestFindEffects(t *testing.T) {
	cases := []struct {
		name      string
		command   string
		cwd       string
		wantKinds []ir.EffectKind
		wantRaws  []string
		wantUnk   []ir.UnknownCode
	}{
		{
			name:      "delete",
			command:   "find . -name '*.txt' -delete",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectDelete},
			wantRaws:  []string{"."},
		},
		{
			name:      "exec unknown",
			command:   "find . -exec rm {} \\;",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectProcess},
			wantRaws:  []string{"find"},
			wantUnk:   []ir.UnknownCode{ir.UnknownEffectsRuntimeDependent},
		},
		{
			name:      "bare read",
			command:   "find src -name '*.go'",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectRead},
			wantRaws:  []string{"src"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractFind(cmd, 0, cond, tc.cwd)
			require.Len(t, effects, len(tc.wantKinds))
			for i, ef := range effects {
				assert.Equal(t, tc.wantKinds[i], ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
			}
			for _, code := range tc.wantUnk {
				assert.True(t, hasUnknownCode(unknowns, code))
			}
		})
	}
}

func TestXargsEffects(t *testing.T) {
	cmd := parseSimple(t, "xargs rm")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	effects, unknowns := effect.ExtractXargs(cmd, 0, cond)
	require.Len(t, effects, 1)
	assert.Equal(t, ir.EffectProcess, effects[0].Kind)
	assert.True(t, hasUnknownCode(unknowns, ir.UnknownEffectsRuntimeDependent))
}

func TestArchiveEffects(t *testing.T) {
	cases := []struct {
		name      string
		command   string
		cwd       string
		wantKinds []ir.EffectKind
		wantRaws  []string
		wantUnk   []ir.UnknownCode
	}{
		{
			name:      "tar extract to dir",
			command:   "tar -xzf archive.tar.gz -C /dest",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"/dest"},
		},
		{
			name:      "tar extract to cwd glob",
			command:   "tar -xzf archive.tar.gz",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"./**"},
			wantUnk:   []ir.UnknownCode{ir.UnknownGlobRuntimeDependent},
		},
		{
			name:      "tar create archive",
			command:   "tar -czf archive.tar.gz files/",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"archive.tar.gz"},
		},
		{
			name:      "unzip to dir",
			command:   "unzip archive.zip -d /out",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"/out"},
		},
		{
			name:      "unzip to cwd glob",
			command:   "unzip archive.zip",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"./**"},
			wantUnk:   []ir.UnknownCode{ir.UnknownGlobRuntimeDependent},
		},
		{
			name:      "zip create archive",
			command:   "zip archive.zip src/",
			cwd:       "/ws",
			wantKinds: []ir.EffectKind{ir.EffectWrite},
			wantRaws:  []string{"archive.zip"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractArchive(cmd, 0, cond, tc.cwd)
			require.Len(t, effects, len(tc.wantKinds))
			for i, ef := range effects {
				assert.Equal(t, tc.wantKinds[i], ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
			}
			for _, code := range tc.wantUnk {
				assert.True(t, hasUnknownCode(unknowns, code))
			}
		})
	}
}

func TestInstallEffectsPackageManagers(t *testing.T) {
	cases := []struct {
		name     string
		command  string
		wantRaws []string
		wantKind ir.EffectKind
	}{
		{
			name:     "pip install",
			command:  "pip install requests",
			wantRaws: []string{"requests"},
			wantKind: ir.EffectInstall,
		},
		{
			name:     "pip3 install",
			command:  "pip3 install -r requirements.txt",
			wantRaws: []string{"."},
			wantKind: ir.EffectInstall,
		},
		{
			name:     "cargo install",
			command:  "cargo install ripgrep",
			wantRaws: []string{"ripgrep"},
			wantKind: ir.EffectInstall,
		},
		{
			name:     "yarn add",
			command:  "yarn add lodash",
			wantRaws: []string{"lodash"},
			wantKind: ir.EffectInstall,
		},
		{
			name:     "bun add",
			command:  "bun add express",
			wantRaws: []string{"express"},
			wantKind: ir.EffectInstall,
		},
		{
			name:     "npx package",
			command:  "npx create-react-app myapp",
			wantRaws: []string{"create-react-app"},
			wantKind: ir.EffectInstall,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractInstall(cmd, 0, cond)
			require.NotEmpty(t, effects, "command=%s", tc.command)
			require.Empty(t, unknowns)
			install := effects[0]
			assert.Equal(t, tc.wantKind, install.Kind)
			assert.Equal(t, tc.wantRaws[0], install.RawTarget)
			// Registry network is possible, never certain.
			require.Len(t, effects, 2, "install + network expected")
			assert.Equal(t, ir.EffectNetwork, effects[1].Kind)
			assert.Equal(t, ir.Possible, effects[1].Certainty)
		})
	}
}

var _ = shell.SimpleCommand{}
