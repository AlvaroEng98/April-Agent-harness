package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// doctorDrift describe un archivo gestionado por el manifiesto que diverge del disco.
type doctorDrift struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "modified" | "missing"
}

// doctorReport es el resultado de `april doctor` — read-only, nunca escribe
// (con la única excepción explícita de `--freeze-baseline`, ver
// runDoctorFreezeBaseline más abajo, que jamás corre como parte de esta
// función ni de la corrida por defecto de `april doctor`).
type doctorReport struct {
	ManifestFound   bool          `json:"manifestFound"`
	ManifestCorrupt bool          `json:"manifestCorrupt"`
	Drifts          []doctorDrift `json:"drifts"`
	Agents          []string      `json:"agents"`
	AgentsInvalid   []string      `json:"agentsInvalid,omitempty"`
	StatusOK        bool          `json:"statusOK"`
	BlockedReasons  []string      `json:"blockedReasons,omitempty"`
	// DebtMetrics — feature 11 (doctor_debt_ratchet): métricas de deuda
	// progresiva ya evaluadas contra su baseline (si hay uno congelado).
	DebtMetrics   []doctorDebtMetric `json:"debtMetrics"`
	UnlinkedTODOs []todoRef          `json:"unlinkedTODOs"`
	Healthy       bool               `json:"healthy"`
}

// computeDoctor calcula el reporte de salud leyendo disco y manifiesto.
// No escribe nada — solo os.ReadFile/os.ReadDir/computeStatus.
func computeDoctor() (doctorReport, error) {
	absTarget, err := filepath.Abs(".")
	if err != nil {
		return doctorReport{}, fmt.Errorf("resolviendo directorio actual: %w", err)
	}

	loaded := loadManifest(absTarget)
	report := doctorReport{
		ManifestFound:   loaded.found,
		ManifestCorrupt: loaded.corrupt,
		Drifts:          []doctorDrift{},
		Agents:          []string{},
	}

	// 1. Drift manifiesto vs disco
	if loaded.corrupt {
		// Manifiesto corrupto: no se puede verificar drift de forma confiable.
		// Se reporta como no saludable.
	} else if loaded.found {
		for relSlash, entry := range loaded.manifest.Files {
			destPath := filepath.Join(absTarget, filepath.FromSlash(relSlash))
			data, err := os.ReadFile(destPath)
			if err != nil {
				if os.IsNotExist(err) {
					report.Drifts = append(report.Drifts, doctorDrift{Path: relSlash, Kind: "missing"})
				} else {
					// Error de lectura distinto: tratar como drift con detalle
					report.Drifts = append(report.Drifts, doctorDrift{Path: relSlash, Kind: "missing"})
				}
				continue
			}
			if hashContent(data) != entry.Hash {
				report.Drifts = append(report.Drifts, doctorDrift{Path: relSlash, Kind: "modified"})
			}
		}
		sort.Slice(report.Drifts, func(i, j int) bool { return report.Drifts[i].Path < report.Drifts[j].Path })
	}
	// Si no hay manifiesto, no hay drift que reportar — modo adopción, igual que scaffold.go

	// 2. Agentes en .claude/agents/
	agentsDir := filepath.Join(absTarget, ".claude", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(agentsDir, e.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				report.AgentsInvalid = append(report.AgentsInvalid, e.Name())
				continue
			}
			if strings.Contains(string(content), "#") {
				report.Agents = append(report.Agents, e.Name())
			} else {
				report.AgentsInvalid = append(report.AgentsInvalid, e.Name())
			}
		}
		sort.Strings(report.Agents)
		sort.Strings(report.AgentsInvalid)
	} else if os.IsNotExist(err) {
		// Sin directorio de agentes: se reporta vacío, no es error fatal por sí solo
	} else {
		return report, fmt.Errorf("leyendo .claude/agents: %w", err)
	}

	// 3. Salud general vía april status
	statusReport, err := computeStatus(nil)
	if err != nil {
		// Si status falla, reportar como no saludable
		report.StatusOK = false
		report.BlockedReasons = []string{fmt.Sprintf("error calculando status: %v", err)}
	} else {
		report.BlockedReasons = statusReport.BlockedReasons
		report.StatusOK = len(statusReport.BlockedReasons) == 0
	}
	if report.BlockedReasons == nil {
		report.BlockedReasons = []string{}
	}
	if report.Drifts == nil {
		report.Drifts = []doctorDrift{}
	}
	if report.Agents == nil {
		report.Agents = []string{}
	}

	// 4. Ratchet de deuda progresiva (feature 11, doctor_debt_ratchet).
	// Solo lectura: el baseline se congela EXCLUSIVAMENTE vía
	// `april doctor --freeze-baseline` (runDoctorFreezeBaseline) — nunca
	// acá, para no romper el contrato read-only de la feature 9. Sin
	// baseline congelado, la métrica se reporta pero nunca hace fallar el
	// comando (nada contra qué medir el crecimiento todavía).
	fsys := os.DirFS(absTarget)
	featureIDs, err := loadKnownFeatureIDs(fsys)
	if err != nil {
		return report, fmt.Errorf("leyendo feature_list.json para métrica de deuda: %w", err)
	}
	unlinkedTODOs, err := computeTODODebt(fsys, featureIDs)
	if err != nil {
		return report, fmt.Errorf("calculando TODOs sin feature asociada: %w", err)
	}
	report.UnlinkedTODOs = unlinkedTODOs
	if report.UnlinkedTODOs == nil {
		report.UnlinkedTODOs = []todoRef{}
	}

	baseline, baselineFrozen, err := loadDoctorBaseline(absTarget)
	if err != nil {
		return report, fmt.Errorf("leyendo baseline de deuda (%s): %w", doctorBaselinePath, err)
	}
	report.DebtMetrics = []doctorDebtMetric{
		evaluateDebtRatchet(todoDebtMetricName, len(unlinkedTODOs), baselineFrozen, baseline.Metrics[todoDebtMetricName]),
	}

	anyDebtExceeded := false
	for _, m := range report.DebtMetrics {
		if m.Exceeded {
			anyDebtExceeded = true
		}
	}

	// Healthy si no hay drift, manifiesto no corrupto, status OK, sin
	// agentes inválidos y sin métrica de deuda que haya crecido respecto a
	// su baseline.
	report.Healthy = len(report.Drifts) == 0 && !report.ManifestCorrupt && report.StatusOK && len(report.AgentsInvalid) == 0 && !anyDebtExceeded
	// Si no hay manifiesto, no se considera no-healthy por eso solo — adopción
	if !report.ManifestFound && !report.ManifestCorrupt {
		// Mantener healthy basado en otros chequeos (agentes + status + deuda)
		report.Healthy = len(report.Drifts) == 0 && report.StatusOK && len(report.AgentsInvalid) == 0 && !anyDebtExceeded
	}

	return report, nil
}

// ---------------------------------------------------------------------
// Feature 11 — doctor_debt_ratchet.
//
// Métrica implementada: cantidad de comentarios marcador-de-pendiente sin
// feature asociada (CHECKPOINTS.md, C3: "No hay TODO sin feature asociada
// en feature_list.json"). Un comentario cuenta como marcador-de-pendiente
// si es de una sola línea y arranca, justo tras "//" y espacios
// opcionales, con esa palabra reservada de cuatro letras en mayúsculas
// (ver todoCommentRe). Cuenta como "sin feature asociada" si además
// ninguna parte de esa línea menciona un id que exista en
// feature_list.json bajo alguno de estos patrones (case-insensitive):
// "feature 7", "feature #7", "#7". Uno que menciona un número que NO
// corresponde a ninguna feature real (p.ej. "feature 999" si esa feature
// no existe) sigue contando como deuda — una referencia a algo
// inexistente no es una asociación válida.
//
// Mecanismo de ratchet: el valor de la métrica se congela en
// .claude/doctor-baseline.json exclusivamente vía el flag explícito
// `april doctor --freeze-baseline` (nunca en la corrida por defecto, ver
// runDoctorFreezeBaseline). Sin baseline congelado, `april doctor` NUNCA
// falla por esta causa — solo informa el valor actual. Con baseline
// congelado, falla (exit != 0, Healthy=false) si la métrica actual supera
// al baseline, señalando nombre de métrica y delta (evaluateDebtRatchet).
// ---------------------------------------------------------------------

// doctorBaselinePath es la ruta (relativa a la raíz del repo) donde
// `april doctor --freeze-baseline` congela el valor de cada métrica de
// deuda. Vive bajo .claude/, igual que .claude/manifest.json, por el
// mismo motivo: es estado de trabajo de ESTE repo (qué tan alta está hoy
// la barra de deuda tolerada), no producto — así que va a .gitignore
// (ver .gitignore, sección "doctor-baseline"). Estar en .gitignore lo
// excluye automáticamente de hashTree/computeSubjectHash (feature 12,
// tree_hash_respects_gitignore) SIN tocar isExcludedFromTreeHash ni
// fixedTreeExclusions — a diferencia de verify-ledger.jsonl (excluido de
// forma incondicional porque CADA verify/review record lo reescribe y
// eso auto-invalidaría el propio treeHash que acaba de registrar), este
// archivo solo cambia por una acción explícita y deliberada
// (--freeze-baseline), nunca como side-effect de cada corrida — así que
// no necesitaba la misma exclusión incondicional, solo no ensuciar el
// árbol versionado con estado de una máquina/sesión particular.
const doctorBaselinePath = ".claude/doctor-baseline.json"

// todoDebtMetricName identifica, dentro de doctorBaseline.Metrics, la
// única métrica que implementa esta feature. El mapa admite más métricas
// a futuro (código muerto, etc.) sin migrar el formato del archivo.
const todoDebtMetricName = "todo_sin_feature"

// doctorBaseline es el contenido persistido de .claude/doctor-baseline.json.
type doctorBaseline struct {
	Metrics map[string]int `json:"metrics"`
}

// todoRef es un TODO sin feature asociada encontrado en el árbol.
type todoRef struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// doctorDebtMetric es una métrica de deuda ya evaluada contra su baseline
// (si lo hay) — lista para el reporte de `april doctor`.
type doctorDebtMetric struct {
	Name           string `json:"name"`
	Current        int    `json:"current"`
	BaselineFrozen bool   `json:"baselineFrozen"`
	Baseline       int    `json:"baseline"`
	Delta          int    `json:"delta"`
	Exceeded       bool   `json:"exceeded"`
}

// todoCommentRe reconoce un comentario que ES el marcador reservado (no
// cualquier aparición de esa palabra en medio de una oración): exige que
// arranque, justo tras "//" y espacios opcionales, con las cuatro
// mayúsculas T-O-D-O. El "^" ancla al inicio del propio token de
// comentario devuelto por go/scanner (nunca a mitad de él), lo que evita
// dos falsos positivos reales de este mismo archivo: (a) status.go tiene
// la línea "// computeBlockedReasons calcula blockedReasons sobre TODO
// feature_list.json", donde esa palabra es el pronombre español "todo" en
// mayúsculas por énfasis, no aparece al inicio del comentario; y (b) los
// comentarios explicativos de esta misma sección citan ese marcador como
// ejemplo textual en medio de una oración, no al inicio. Cubre marcadores
// al inicio de línea y también después de código en la misma línea; no
// cubre bloques /* */ (el repo no los usa, ver docs/conventions.md
// "Comentarios").
var todoCommentRe = regexp.MustCompile(`^//\s*TODO\b`)

// featureRefRe encuentra referencias a un id de feature dentro del texto
// de un TODO: "feature 7", "feature #7", o "#7" (case-insensitive). El
// número capturado (grupo 1 o 2, el que no esté vacío) se valida contra
// los ids reales de feature_list.json en isTODOLinkedToFeature.
var featureRefRe = regexp.MustCompile(`(?i)feature\s*#?\s*(\d+)|#(\d+)`)

// isTODOLinkedToFeature decide si el texto de un TODO menciona un id de
// feature que existe en featureIDs.
func isTODOLinkedToFeature(text string, featureIDs map[int]bool) bool {
	for _, m := range featureRefRe.FindAllStringSubmatch(text, -1) {
		numStr := m[1]
		if numStr == "" {
			numStr = m[2]
		}
		n, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if featureIDs[n] {
			return true
		}
	}
	return false
}

// findUnlinkedTODOsInContent tokeniza content como código Go (go/scanner,
// stdlib) y devuelve los comentarios TODO que no mencionan ningún id de
// feature existente. Pura: no toca disco.
//
// Tokenizar en vez de escanear línea por línea evita un falso positivo
// real: un `regexp` sobre el texto crudo de la línea también "encuentra"
// un TODO dentro de un literal de string Go (p.ej. los propios fixtures
// de doctor_test.go, que contienen el texto `"// TODO: ..."` como dato de
// prueba, no como comentario real). go/scanner distingue el token COMMENT
// del token STRING aunque el contenido textual sea idéntico. No exige que
// content sea un archivo .go sintácticamente completo: go/scanner
// tokeniza léxicamente, sin validar gramática, así que fragmentos de
// prueba sueltos también se tokenizan sin problema.
func findUnlinkedTODOsInContent(path, content string, featureIDs map[int]bool) []todoRef {
	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(content))

	var s scanner.Scanner
	s.Init(file, []byte(content), nil, scanner.ScanComments)

	var found []todoRef
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			continue
		}
		if !todoCommentRe.MatchString(lit) {
			continue
		}
		if isTODOLinkedToFeature(lit, featureIDs) {
			continue
		}
		found = append(found, todoRef{Path: path, Line: fset.Position(pos).Line, Text: strings.TrimSpace(lit)})
	}
	return found
}

// computeTODODebt recorre fsys buscando archivos *.go (excluye .git/,
// igual que hashTree) y agrega findUnlinkedTODOsInContent sobre cada uno.
// Pura sobre fs.FS — mismo patrón de dos capas que hashTree/
// computeStatusFromFS.
func computeTODODebt(fsys fs.FS, featureIDs map[int]bool) ([]todoRef, error) {
	var all []todoRef
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		all = append(all, findUnlinkedTODOsInContent(p, string(data), featureIDs)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		return all[i].Line < all[j].Line
	})
	return all, nil
}

// loadKnownFeatureIDs lee feature_list.json de fsys y devuelve el
// conjunto de ids existentes — usado para decidir si un TODO está
// asociado a una feature real. Reusa featureListFile/featureEntry
// (status.go) en vez de duplicar el parseo. feature_list.json ausente no
// es un error acá (mismo criterio de adopción que el resto de doctor.go):
// simplemente ningún TODO puede estar asociado a nada.
func loadKnownFeatureIDs(fsys fs.FS) (map[int]bool, error) {
	data, err := fs.ReadFile(fsys, featureListPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[int]bool{}, nil
		}
		return nil, fmt.Errorf("leyendo %s: %w", featureListPath, err)
	}
	var fl featureListFile
	if err := json.Unmarshal(data, &fl); err != nil {
		return nil, fmt.Errorf("parseando %s: %w", featureListPath, err)
	}
	ids := make(map[int]bool, len(fl.Features))
	for _, f := range fl.Features {
		ids[f.ID] = true
	}
	return ids, nil
}

// parseDoctorBaseline interpreta el contenido ya leído de
// .claude/doctor-baseline.json. Contenido vacío (archivo inexistente,
// tratado como adopción por el wrapper loadDoctorBaseline) devuelve un
// baseline con Metrics vacío, no un error.
func parseDoctorBaseline(data []byte) (doctorBaseline, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return doctorBaseline{Metrics: map[string]int{}}, nil
	}
	var b doctorBaseline
	if err := json.Unmarshal(data, &b); err != nil {
		return doctorBaseline{}, fmt.Errorf("parseando %s: %w", doctorBaselinePath, err)
	}
	if b.Metrics == nil {
		b.Metrics = map[string]int{}
	}
	return b, nil
}

// loadDoctorBaseline es el wrapper de I/O de parseDoctorBaseline: lee
// .claude/doctor-baseline.json bajo absTarget. Archivo inexistente no es
// error — found=false, mismo criterio de adopción que loadManifest
// (scaffold.go).
func loadDoctorBaseline(absTarget string) (baseline doctorBaseline, found bool, err error) {
	data, err := os.ReadFile(filepath.Join(absTarget, doctorBaselinePath))
	if err != nil {
		if os.IsNotExist(err) {
			return doctorBaseline{Metrics: map[string]int{}}, false, nil
		}
		return doctorBaseline{}, false, fmt.Errorf("leyendo %s: %w", doctorBaselinePath, err)
	}
	b, err := parseDoctorBaseline(data)
	if err != nil {
		return doctorBaseline{}, true, err
	}
	return b, true, nil
}

// writeDoctorBaseline persiste b en .claude/doctor-baseline.json bajo
// absTarget, de forma atómica (writeFileAtomic, set_status.go). Único
// punto de escritura de esta feature — invocado solo por
// runDoctorFreezeBaseline (--freeze-baseline), jamás por la corrida por
// defecto de `april doctor` (contrato read-only de la feature 9,
// doctor_readonly_check).
func writeDoctorBaseline(absTarget string, b doctorBaseline) error {
	dir := filepath.Join(absTarget, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creando %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando %s: %w", doctorBaselinePath, err)
	}
	data = append(data, '\n')
	return writeFileAtomic(filepath.Join(absTarget, doctorBaselinePath), data, 0644)
}

// evaluateDebtRatchet compara current contra baseline — pura, sin I/O.
// Sin baseline congelado (baselineFrozen=false) nunca excede: recién a
// partir de que exista un baseline hay algo contra qué medir el
// crecimiento (acceptance de la feature 11, "primera corrida sin
// baseline ... no falla por deuda existente").
func evaluateDebtRatchet(name string, current int, baselineFrozen bool, baseline int) doctorDebtMetric {
	m := doctorDebtMetric{Name: name, Current: current, BaselineFrozen: baselineFrozen}
	if !baselineFrozen {
		return m
	}
	m.Baseline = baseline
	m.Delta = current - baseline
	m.Exceeded = current > baseline
	return m
}

// runDoctorFreezeBaseline es la ÚNICA vía de escritura de esta feature:
// `april doctor --freeze-baseline` calcula el valor actual de cada
// métrica de deuda y lo persiste en .claude/doctor-baseline.json. Es la
// excepción explícita y deliberada al contrato read-only de `april
// doctor` fijado por la feature 9 (doctor_readonly_check) — se activa
// SOLO con este flag; la corrida por defecto (con o sin --json) sigue
// siendo 100% read-only. Si ya existe un baseline congelado, se niega a
// sobreescribirlo en silencio — el ratchet solo debe ceder de forma
// explícita: hay que borrar .claude/doctor-baseline.json a mano si de
// verdad se quiere recongelar tras aceptar deuda nueva a propósito.
func runDoctorFreezeBaseline() int {
	absTarget, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	_, found, err := loadDoctorBaseline(absTarget)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if found {
		fmt.Fprintf(os.Stderr, "Error: ya existe un baseline congelado en %s — borralo a mano si de verdad querés recongelarlo\n", doctorBaselinePath)
		return 1
	}

	fsys := os.DirFS(absTarget)
	featureIDs, err := loadKnownFeatureIDs(fsys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	unlinkedTODOs, err := computeTODODebt(fsys, featureIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	newBaseline := doctorBaseline{Metrics: map[string]int{todoDebtMetricName: len(unlinkedTODOs)}}
	if err := writeDoctorBaseline(absTarget, newBaseline); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("[OK] Baseline de deuda congelado en %s: %s = %d\n", doctorBaselinePath, todoDebtMetricName, len(unlinkedTODOs))
	return 0
}

// runDoctor ejecuta `april doctor` y devuelve exit code.
func runDoctor(args []string) int {
	jsonOutput := false
	for _, a := range args {
		if a == "--freeze-baseline" {
			return runDoctorFreezeBaseline()
		}
		if a == "--json" {
			jsonOutput = true
		}
	}

	report, err := computeDoctor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(string(data))
		if report.Healthy {
			return 0
		}
		return 1
	}

	// Salida en texto plano
	fmt.Println("── april doctor ───────────────────────────")
	if report.ManifestCorrupt {
		fmt.Println("[FAIL] .claude/manifest.json corrupto (JSON inválido)")
	} else if !report.ManifestFound {
		fmt.Println("[WARN] .claude/manifest.json no encontrado (modo adopción — sin drift que verificar)")
	} else if len(report.Drifts) == 0 {
		fmt.Println("[OK]   Manifiesto sin drift")
	} else {
		fmt.Printf("[FAIL] %d drift(s) detectado(s):\n", len(report.Drifts))
		for _, d := range report.Drifts {
			fmt.Printf("  - %s (%s)\n", d.Path, d.Kind)
		}
	}

	if len(report.AgentsInvalid) > 0 {
		for _, a := range report.AgentsInvalid {
			fmt.Printf("[FAIL] Agente sin cabecera válida: %s\n", a)
		}
	}
	if len(report.Agents) == 0 && len(report.AgentsInvalid) == 0 {
		fmt.Println("[WARN] No se encontraron agentes en .claude/agents/")
	} else if len(report.Agents) > 0 {
		for _, a := range report.Agents {
			fmt.Printf("[OK]   Agente válido: %s\n", a)
		}
		fmt.Printf("[OK]   %d agente(s) encontrado(s)\n", len(report.Agents))
	}

	if report.StatusOK {
		fmt.Println("[OK]   april status sin blockedReasons")
	} else {
		fmt.Println("[FAIL] april status reportó blockedReasons:")
		for _, r := range report.BlockedReasons {
			fmt.Printf("  - %s\n", r)
		}
	}

	for _, m := range report.DebtMetrics {
		if !m.BaselineFrozen {
			fmt.Printf("[WARN] Métrica de deuda %q sin baseline congelado (actual: %d) — correr `april doctor --freeze-baseline` para empezar a exigir que no crezca\n", m.Name, m.Current)
		} else if m.Exceeded {
			fmt.Printf("[FAIL] Métrica de deuda %q creció: actual %d > baseline %d (delta +%d)\n", m.Name, m.Current, m.Baseline, m.Delta)
		} else {
			fmt.Printf("[OK]   Métrica de deuda %q dentro del baseline (actual %d, baseline %d, delta %d)\n", m.Name, m.Current, m.Baseline, m.Delta)
		}
	}

	fmt.Println("──────────────────────────────────────────")
	if report.Healthy {
		fmt.Println("[OK] Entorno saludable.")
		return 0
	}
	fmt.Println("[FAIL] Entorno con incidencias.")
	return 1
}

// cmdDoctor es el entry point del CLI para `april doctor`.
func cmdDoctor() {
	os.Exit(runDoctor(os.Args[2:]))
}
