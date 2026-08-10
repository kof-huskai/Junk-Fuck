package classifier

import (
	"testing"

	"github.com/kof-huskai/Junk-Fuck/internal/model"
)

func TestFileExtensions(t *testing.T) {
	rs := New()
	cases := map[string]model.Category{
		"test.tmp":         model.CategoryTempFiles,
		"cache.temp":       model.CategoryTempFiles,
		"data.cache":       model.CategoryTempFiles,
		"app.log":          model.CategoryLogs,
		"events.etl":       model.CategoryLogs,
		"crash.dmp":        model.CategoryCrashDumps,
		"memory.dump":      model.CategoryCrashDumps,
		"file.bak":         model.CategoryBackups,
		"old.backup":       model.CategoryBackups,
		"dl.part":          model.CategoryPartialDownloads,
		"file.crdownload":  model.CategoryPartialDownloads,
		"module.pyc":       model.CategoryBuildArtifacts,
		"module.o":         model.CategoryBuildArtifacts,
		"scratch.swp":      model.CategoryEditorTemp,
		"thumb.thumbcache": model.CategoryCache,
		"notes.txt~":       model.CategoryTempFiles,
		"weird.lock":       model.CategoryOtherJunk,
		"leftover.chk":     model.CategoryOtherJunk,
	}
	for name, want := range cases {
		m := rs.Classify(name, false)
		if !m.Matched || m.Category != want {
			t.Errorf("%s: got matched=%v cat=%s want cat=%s", name, m.Matched, m.Category, want)
		}
	}
}

func TestSafeFilesAreNotClassified(t *testing.T) {
	rs := New()
	safe := []string{
		"normal_document.txt",
		"notes.docx",
		"photo.png",
		"report.pdf",
		"video.mp4",
		"data.xlsx",
		"archive.zip",
		"setup.exe",
		"template.docx", // contains "temp" as substring only — must NOT match
		"old_report.md", // broad "old" keyword was dropped — must NOT match
		"copy_of_notes.txt",
	}
	for _, name := range safe {
		if m := rs.Classify(name, false); m.Matched {
			t.Errorf("%s: should not be classified (got %s: %s)", name, m.Category, m.Reason)
		}
	}
}

func TestConservativeTokens(t *testing.T) {
	rs := New()
	cases := map[string]model.Category{
		"my_temp_notes.txt":  model.CategoryTempFiles,
		"report.tmp_v2.xlsx": model.CategoryTempFiles, // token tmp_v2 -> tokens [tmp v2]
		"app_cache_data.bin": model.CategoryCache,
		"db_backup.tar":      model.CategoryBackups,
	}
	for name, want := range cases {
		m := rs.Classify(name, false)
		if !m.Matched || m.Category != want {
			t.Errorf("%s: got matched=%v cat=%s want cat=%s", name, m.Matched, m.Category, want)
		}
	}
}

func TestFolders(t *testing.T) {
	rs := New()
	cases := map[string]model.Category{
		"cache":                    model.CategoryCache,
		"Cache":                    model.CategoryCache,
		"temp":                     model.CategoryTempFiles,
		"Temp":                     model.CategoryTempFiles,
		"logs":                     model.CategoryLogs,
		"__pycache__":              model.CategoryBuildArtifacts,
		"thumbnails":               model.CategoryCache,
		"Prefetch":                 model.CategoryOtherJunk,
		"trash":                    model.CategoryOtherJunk,
		"Downloaded Installations": model.CategoryOtherJunk,
	}
	for name, want := range cases {
		m := rs.Classify(name, true)
		if !m.Matched || m.Category != want {
			t.Errorf("%s: got matched=%v cat=%s want cat=%s", name, m.Matched, m.Category, want)
		}
	}
	safeDirs := []string{"Documents", "MyMusic", "Projects", "normal_folder", "tempfiles"} // "tempfiles" is not an exact rule
	for _, name := range safeDirs {
		if m := rs.Classify(name, true); m.Matched {
			t.Errorf("dir %s: should not be classified (got %s)", name, m.Category)
		}
	}
}

func TestReasonIsInformative(t *testing.T) {
	rs := New()
	m := rs.Classify("test.tmp", false)
	if !m.Matched || m.Reason == "" {
		t.Fatalf("expected an informative reason, got %q", m.Reason)
	}
}
