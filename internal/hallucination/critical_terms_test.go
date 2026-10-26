package hallucination

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vasya/protocol-voice-assistant/internal/logger"
)

func initLog(t *testing.T) {
	t.Helper()
	_ = logger.Init(logger.Config{LogDir: t.TempDir(), LogFile: "t.log", MaxBytes: 1 << 20, KeepOld: 2})
}

func TestLoad(t *testing.T) {
	initLog(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "terms.json")
	os.WriteFile(path, []byte(`{
		"critical_terms": ["перелом", "пневмоторакс", "очаг"]
	}`), 0o644)

	te, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(te.List) != 3 {
		t.Errorf("терминов = %d, want 3", len(te.List))
	}
}

func TestLoadMissing(t *testing.T) {
	initLog(t)
	te, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil {
		t.Errorf("отсутствующий файл не ошибка: %v", err)
	}
	if te == nil || len(te.List) != 0 {
		t.Errorf("должен быть пустой Terms")
	}
}

func TestCheckFound(t *testing.T) {
	initLog(t)
	te := &Terms{List: []string{"перелом", "киста"}}
	rep := te.Check("перелом диафиза, киста", "Обнаружен перелом диафиза; киста без признаков активности")
	if !rep.CheckPassed {
		t.Errorf("Check должен пройти: %+v", rep)
	}
	if len(rep.TermsMissing) != 0 {
		t.Errorf("missing = %v", rep.TermsMissing)
	}
}

func TestCheckMissing(t *testing.T) {
	initLog(t)
	te := &Terms{List: []string{"перелом", "пневмоторакс"}}
	// в диктовке было "перелом", в ответе — только "перелом", пневмоторакс в диктовке отсутствует → не влияет
	rep := te.Check("перелом и пневмоторакс", "Описана линия перелома")
	if rep.CheckPassed {
		t.Errorf("Check должен провалиться: диктовка с пневмотораксом, ответ без него")
	}
	if !slices.Contains(rep.TermsMissing, "пневмоторакс") {
		t.Errorf("пневмоторакс должен быть в missing: %v", rep.TermsMissing)
	}
}

func TestContainsLexicallyPrefix(t *testing.T) {
	// русское окончание: "перелома" содержит префикс "перелом"
	if !containsLexically("описан перелом диафиза", "перелом") {
		t.Error("точное вхождение не найдено")
	}
	if !containsLexically("видна линия перелома", "перелом") {
		t.Error("префиксное окончание не найдено")
	}
	if containsLexically("нет перелома здесь нет", "пневмоторакс") {
		t.Error("не должно находить отсутствующий термин")
	}
}
