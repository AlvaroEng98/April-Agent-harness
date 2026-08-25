package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Se embeben el tooling (idéntico al que dogfoodea este repo) y el directorio
// templates/, que contiene el LIENZO LIMPIO de los archivos con estado
// (feature_list.json, progress/, docs/, .gitignore). El estado de trabajo del
// propio harness vive en la raíz (feature_list.json, progress/, docs/,
// .gitignore) y NO se embebe: así cada `harness init` genera un proyecto en
// limpio, no la bitácora ni las reglas de ignore del harness. El .gitignore
// de la raíz de este repo tiene reglas propias del desarrollo del harness
// (OS, IDE, build de Go, notas de sesión) que no aplican al proyecto
// scaffoldeado; por eso el template vive aparte, en templates/.gitignore, con
// solo las reglas que sí aplican al destino.
//
// release-notes.sh, sync-changelog.sh y .goreleaser.yaml NO se embeben: son
// del pipeline de release de este repo (Go + goreleaser), no del arnés que se
// scaffoldea. CHANGELOG.md tampoco tiene plantilla: el proyecto scaffoldeado
// no arranca con changelog propio.
//
//go:embed .claude AGENTS.md CLAUDE.md init.sh session-handoff.md CHECKPOINTS.md all:templates
var templateFS embed.FS

// manifestSchemaVersion es la versión del formato de .claude/manifest.json.
// Sube si el formato del manifiesto cambia de forma incompatible.
const manifestSchemaVersion = 1

// manifestEntry registra el hash con el que april dejó un archivo la última
// vez que escribió sobre el destino.
type manifestEntry struct {
	Hash string `json:"hash"`
}

// manifest es el "last-applied-configuration" de april init sobre un
// destino: ruta relativa (forward-slash) → hash sha256 del contenido con el
// que april dejó ese archivo la última vez. Permite en la siguiente corrida
// diferenciar "el usuario no tocó esto, sincroniza con la plantilla nueva"
// de "el usuario lo modificó, no lo pierdas" sin trato hardcodeado por
// nombre de archivo. .gitignore queda fuera a propósito: tiene su propia
// lógica de merge idempotente (mergeGitignore) que ya resuelve el caso solo.
type manifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Files         map[string]manifestEntry `json:"files"`
}

// manifestLoadResult envuelve el resultado de loadManifest: el manifiesto
// leído (vacío si no existía o estaba corrupto), si existía físicamente
// (found) y si su JSON era inválido (corrupt). Ninguno de los dos casos es
// un error fatal: ambos son la señal para que el llamador entre en modo
// adopción en vez de abortar el comando.
type manifestLoadResult struct {
	manifest manifest
	found    bool
	corrupt  bool
}

// hashContent calcula el sha256 hexadecimal de content: es la huella que
// registra el manifiesto para saber "con qué contenido dejó april este
// archivo".
func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// manifestPath devuelve la ruta absoluta de .claude/manifest.json dentro de
// absTarget.
func manifestPath(absTarget string) string {
	return filepath.Join(absTarget, ".claude", "manifest.json")
}

// loadManifest lee el manifiesto anterior de absTarget. Es de solo lectura y
// nunca falla de forma fatal: si el archivo no existe (primera corrida) o su
// JSON es inválido (manifiesto corrupto), devuelve un manifiesto vacío con
// found/corrupt reflejando cuál de los dos casos fue, para que el llamador
// entre en modo adopción en vez de abortar.
func loadManifest(absTarget string) manifestLoadResult {
	empty := manifest{SchemaVersion: manifestSchemaVersion, Files: map[string]manifestEntry{}}

	data, err := os.ReadFile(manifestPath(absTarget))
	if err != nil {
		return manifestLoadResult{manifest: empty}
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return manifestLoadResult{manifest: empty, corrupt: true}
	}
	if m.Files == nil {
		m.Files = map[string]manifestEntry{}
	}
	return manifestLoadResult{manifest: m, found: true}
}

// writeManifest escribe el manifiesto actualizado en absTarget tras aplicar
// el plan, para que la siguiente corrida de april init pueda diferenciar
// archivos aún sincronizados con la plantilla de archivos tocados por el
// usuario.
func writeManifest(absTarget string, m manifest) error {
	m.SchemaVersion = manifestSchemaVersion
	if m.Files == nil {
		m.Files = map[string]manifestEntry{}
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	path := manifestPath(absTarget)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// fileAction es la decisión tomada para un archivo concreto al scaffoldear,
// resultado de comparar disco, manifiesto anterior y plantilla nueva (ver la
// tabla de decisión en el godoc de planScaffoldFromFS).
type fileAction int

const (
	// actionCreate: el archivo no estaba en el manifiesto anterior (es nuevo
	// del template, o no hay manifiesto porque es la primera corrida). Se
	// escribe y se registra.
	actionCreate fileAction = iota
	// actionUpdate: el disco coincide con el hash del manifiesto anterior (el
	// usuario no tocó el archivo). Se sobreescribe con la plantilla nueva y
	// se actualiza el hash.
	actionUpdate
	// actionSkipUnmodified: el usuario tocó el archivo pero la plantilla no
	// cambió respecto de lo registrado. Se deja tal cual, sin aviso.
	actionSkipUnmodified
	// actionSkipConflict: el usuario tocó el archivo y la plantilla también
	// cambió (conflicto real). Se conserva la versión del usuario y se avisa.
	actionSkipConflict
	// actionAdopt: no hay manifiesto previo válido (primera corrida tras esta
	// mejora, o manifiesto corrupto). El archivo ya existe en disco: no se
	// toca, solo se adopta su hash actual como línea base.
	actionAdopt
)

// scaffoldFileDelete describe un archivo que ya no viene en la plantilla
// nueva y que se borra porque el disco coincide con el hash que tenía
// registrado en el manifiesto anterior (el usuario no lo modificó).
type scaffoldFileDelete struct {
	relPath  string
	destPath string
}

// scaffoldFileWrite describe un único archivo a escribir (o no) en el
// destino: su ruta relativa (para el mensaje de progreso y el manifiesto),
// su ruta absoluta de destino, el contenido ya resuelto (incluido el merge
// de .gitignore si aplica), el modo con el que se escribe, la acción
// decidida por la tabla de decisión y el hash con el que queda registrado en
// el manifiesto tras aplicar el plan.
type scaffoldFileWrite struct {
	relPath  string
	destPath string
	content  []byte
	mode     fs.FileMode
	action   fileAction
	newHash  string
}

// scaffoldPlan es el resultado, ya decidido, de qué hacer al scaffoldear
// absTarget: qué escribir, qué borrar y con qué acción, el manifiesto
// resultante a persistir, y qué directorios vacíos crear. No contiene
// ninguna llamada de I/O de escritura en sí mismo: lo produce planScaffold
// (puro) y lo consume applyPlan (ejecutor).
type scaffoldPlan struct {
	absTarget       string
	createTargetDir bool
	dirsToCreate    []string
	files           []scaffoldFileWrite
	filesToDelete   []scaffoldFileDelete
	// obsoleteConflicts son rutas relativas que ya no vienen en la plantilla
	// nueva pero se conservan porque el usuario las modificó (no coinciden
	// con el hash del manifiesto anterior): applyPlan avisa por cada una.
	obsoleteConflicts []string
	emptyDirs         []string
	// manifest es el manifiesto que applyPlan debe persistir al terminar:
	// refleja el estado resultante de aplicar este plan, no el anterior.
	manifest manifest
	// manifestFound/manifestCorrupt describen el manifiesto ANTERIOR leído
	// por planScaffoldFromFS, para que applyPlan pueda avisar de un modo
	// adopción (ausente o corrupto) antes de aplicar el resto.
	manifestFound   bool
	manifestCorrupt bool
}

// planScaffold decide todo lo que hay que hacer para scaffoldear absTarget
// sin tocar el disco (salvo lecturas), usando el embed.FS real de producción.
func planScaffold(absTarget string) (scaffoldPlan, error) {
	return planScaffoldFromFS(absTarget, templateFS)
}

// planScaffoldFromFS decide todo lo que hay que hacer para scaffoldear
// absTarget contra tmplFS sin tocar el disco (salvo lecturas): recorre el
// fs.FS de plantilla, compara cada archivo contra el manifiesto anterior
// (.claude/manifest.json) y el contenido en disco, y aplica esta tabla de
// decisión (mismo patrón que el three-way merge de `kubectl apply`):
//
//	disco == manifiesto | plantilla == manifiesto | Acción
//	(no estaba en manifiesto anterior) | —      | create
//	sí (usuario no tocó)               | —      | update
//	no (usuario tocó)                  | sí     | skip silencioso (unmodified)
//	no (usuario tocó)                  | no     | skip con aviso (conflict)
//
// Los archivos que estaban en el manifiesto anterior pero ya no vienen en la
// plantilla nueva se borran solo si el disco coincide con el hash registrado
// (usuario no los tocó); si no coinciden, se conservan y se avisa.
//
// .gitignore queda FUERA de este mecanismo por completo: mantiene su rama
// actual de merge (mergeGitignore), que ya es idempotente y no necesita el
// diff genérico.
//
// Si no hay manifiesto anterior válido (no existe, o su JSON es inválido),
// se entra en modo adopción: no se sobreescribe ni se borra nada que ya
// exista en disco, solo se crean los archivos de plantilla que falten y se
// adopta el hash de lo existente como línea base para la corrida siguiente.
//
// Se extrae de planScaffold como wrapper parametrizado por el fs.FS de
// plantilla para poder inyectar un fstest.MapFS sintético en los tests que
// simulan "la plantilla cambió" o "la plantilla ya no trae este archivo" sin
// depender del embed.FS real. Sigue siendo pura: solo os.ReadDir/os.ReadFile/
// os.Stat, nunca escribe.
func planScaffoldFromFS(absTarget string, tmplFS fs.FS) (scaffoldPlan, error) {
	plan := scaffoldPlan{absTarget: absTarget}

	if _, err := os.ReadDir(absTarget); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return scaffoldPlan{}, err
		}
		plan.createTargetDir = true
	}

	loaded := loadManifest(absTarget)
	plan.manifestFound = loaded.found
	plan.manifestCorrupt = loaded.corrupt
	prevManifest := loaded.manifest
	// Sin manifiesto anterior válido (ausente o corrupto) no hay línea base
	// contra la que diffear: se adopta lo que ya exista en disco en vez de
	// arriesgarse a sobreescribir o borrar trabajo del usuario.
	adopting := !loaded.found || loaded.corrupt

	newManifest := manifest{SchemaVersion: manifestSchemaVersion, Files: map[string]manifestEntry{}}
	seen := map[string]bool{}

	err := fs.WalkDir(tmplFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		// templates/ contiene el lienzo limpio de los archivos con estado.
		// Se copia a la raíz del destino stripping el prefijo "templates/".
		// El directorio "templates" en sí no se crea en el destino.
		relPath := path
		if path == "templates" {
			return nil
		}
		if strings.HasPrefix(path, "templates/") {
			relPath = strings.TrimPrefix(path, "templates/")
		}

		// todos los archivos y directorios (incluido el árbol .claude/) se
		// copian tal cual, sin transformación
		destPath := filepath.Join(absTarget, relPath)
		if d.IsDir() {
			plan.dirsToCreate = append(plan.dirsToCreate, destPath)
			return nil
		}

		relSlash := filepath.ToSlash(relPath)

		// Guard de dogfooding: si el propio .claude/manifest.json de este
		// repo quedara embebido en tmplFS (por correr april init sobre este
		// mismo repo por error), nunca se propaga a un destino.
		if relSlash == ".claude/manifest.json" {
			return nil
		}

		data, err := fs.ReadFile(tmplFS, path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0644)
		switch d.Name() {
		case "init.sh":
			mode = 0755
		}
		fw := scaffoldFileWrite{
			relPath:  relSlash,
			destPath: destPath,
			mode:     mode,
		}

		// .gitignore queda fuera del manifiesto: sigue con su lógica de merge
		// propia, ya idempotente, en vez del diff genérico de abajo.
		if relPath == ".gitignore" {
			if _, err := os.Stat(destPath); err == nil {
				merged, err := mergeGitignore(destPath, data)
				if err != nil {
					return err
				}
				fw.content = merged
				fw.action = actionUpdate
				plan.files = append(plan.files, fw)
				return nil
			}
			fw.content = data
			fw.action = actionCreate
			plan.files = append(plan.files, fw)
			return nil
		}

		seen[relSlash] = true
		templateHash := hashContent(data)

		diskData, diskErr := os.ReadFile(destPath)
		diskExists := diskErr == nil

		prevEntry, hadPrev := prevManifest.Files[relSlash]

		switch {
		case adopting:
			if diskExists {
				// No se toca: se adopta el hash de lo que ya hay en disco
				// como línea base para que la corrida siguiente sí diffee.
				fw.action = actionAdopt
				fw.newHash = hashContent(diskData)
				fw.content = diskData
			} else {
				fw.action = actionCreate
				fw.newHash = templateHash
				fw.content = data
			}
		case !hadPrev:
			// Archivo nuevo del template: no había línea base que proteger.
			fw.action = actionCreate
			fw.newHash = templateHash
			fw.content = data
		case !diskExists:
			// El manifiesto lo registraba pero ya no está en disco (el
			// usuario lo borró sin dejar rastro de edición): se recrea desde
			// la plantilla, igual que si no lo hubiera tocado.
			fw.action = actionUpdate
			fw.newHash = templateHash
			fw.content = data
		case hashContent(diskData) == prevEntry.Hash:
			// Disco == manifiesto: el usuario no tocó el archivo, sincroniza.
			fw.action = actionUpdate
			fw.newHash = templateHash
			fw.content = data
		case prevEntry.Hash == templateHash:
			// Usuario tocó el archivo, la plantilla no cambió: skip
			// silencioso, se deja tal cual.
			fw.action = actionSkipUnmodified
			fw.newHash = hashContent(diskData)
			fw.content = diskData
		default:
			// Usuario tocó el archivo Y la plantilla también cambió:
			// conflicto real, se conserva la versión del usuario y se avisa.
			fw.action = actionSkipConflict
			fw.newHash = hashContent(diskData)
			fw.content = diskData
		}

		newManifest.Files[relSlash] = manifestEntry{Hash: fw.newHash}
		plan.files = append(plan.files, fw)
		return nil
	})
	if err != nil {
		return scaffoldPlan{}, err
	}

	// Archivos que estaban en el manifiesto anterior pero ya no vienen en la
	// plantilla nueva: se borran solo si el disco coincide con el hash
	// registrado (usuario no los tocó); si no coinciden, se conservan y se
	// avisa. En modo adopción prevManifest.Files está vacío (no hay línea
	// base), así que este bucle no encuentra nada que borrar.
	for relSlash, entry := range prevManifest.Files {
		if seen[relSlash] {
			continue
		}
		destPath := filepath.Join(absTarget, filepath.FromSlash(relSlash))
		diskData, diskErr := os.ReadFile(destPath)
		if diskErr != nil {
			// ya no existe en disco tampoco: nada que borrar ni avisar.
			continue
		}
		if hashContent(diskData) == entry.Hash {
			plan.filesToDelete = append(plan.filesToDelete, scaffoldFileDelete{relPath: relSlash, destPath: destPath})
		} else {
			plan.obsoleteConflicts = append(plan.obsoleteConflicts, relSlash)
		}
	}

	plan.manifest = newManifest
	plan.emptyDirs = []string{
		filepath.Join(absTarget, "specs"),
	}

	return plan, nil
}

// applyPlan ejecuta un scaffoldPlan ya decidido: solo hace las llamadas
// os.WriteFile/os.Remove/os.MkdirAll, imprime los avisos correspondientes a
// cada acción y persiste el manifiesto resultante. No toma ninguna decisión
// de contenido: eso ya lo resolvió planScaffoldFromFS.
func applyPlan(plan scaffoldPlan) error {
	if plan.createTargetDir {
		if err := os.MkdirAll(plan.absTarget, 0755); err != nil {
			return err
		}
	}

	switch {
	case plan.manifestCorrupt:
		fmt.Println("  Aviso: .claude/manifest.json existente es corrupto (JSON inválido); se reconstruye en modo adopción sin sobreescribir ni borrar nada existente.")
	case !plan.manifestFound:
		fmt.Println("  No se encontró .claude/manifest.json (proyecto scaffoldeado con una versión anterior de april); modo adopción: se adopta el contenido existente como línea base, sin sobreescribir ni borrar nada.")
	}

	for _, dir := range plan.dirsToCreate {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	created, updated := 0, 0
	for _, fw := range plan.files {
		switch fw.action {
		case actionCreate:
			if err := os.WriteFile(fw.destPath, fw.content, fw.mode); err != nil {
				return err
			}
			created++
		case actionUpdate:
			if err := os.WriteFile(fw.destPath, fw.content, fw.mode); err != nil {
				return err
			}
			updated++
		case actionAdopt, actionSkipUnmodified:
			// nada que escribir: el disco ya tiene el contenido correcto.
		case actionSkipConflict:
			fmt.Printf("  Aviso: %s fue modificado y la plantilla también cambió; se conserva tu versión. Revisa manualmente la plantilla nueva.\n", fw.relPath)
		}
	}

	for _, del := range plan.filesToDelete {
		if err := os.Remove(del.destPath); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: no se pudo borrar %s: %v\n", del.relPath, err)
		}
	}

	for _, relPath := range plan.obsoleteConflicts {
		fmt.Printf("  Aviso: %s ya no forma parte de la plantilla, pero fue modificado; se conserva.\n", relPath)
	}

	for _, dirPath := range plan.emptyDirs {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create %s/\n", filepath.ToSlash(dirPath))
		} else {
			created++
		}
	}

	switch {
	case updated > 0 && created > 0:
		fmt.Printf("  %d files created, %d updated\n", created, updated)
	case updated > 0:
		fmt.Printf("  %d files updated\n", updated)
	default:
		fmt.Printf("  %d files created\n", created)
	}

	fmt.Println()
	fmt.Printf("Done! Harness project scaffolded in %s\n", plan.absTarget)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit feature_list.json with your project info")
	fmt.Println("  2. Run ./init.sh to verify the environment")
	fmt.Println("  3. Read AGENTS.md to understand the workflow")

	return writeManifest(plan.absTarget, plan.manifest)
}

// scaffoldInit contiene la lógica de scaffolding de `harness init` sobre un
// directorio destino ya resuelto a ruta absoluta. Se separa de cmdInit (que
// depende de os.Args y de os.Exit) para poder testearla in-process contra un
// directorio temporal. Es pura composición: decide con planScaffold y
// ejecuta con applyPlan.
func scaffoldInit(absTarget string) error {
	plan, err := planScaffold(absTarget)
	if err != nil {
		return err
	}
	return applyPlan(plan)
}

// mergeGitignore agrega al archivo existente las entradas del template que falten.
func mergeGitignore(existingPath string, templateData []byte) ([]byte, error) {
	existing, err := os.ReadFile(existingPath)
	if err != nil {
		return nil, err
	}

	existingLines := strings.Split(string(existing), "\n")
	existingSet := make(map[string]bool, len(existingLines))
	for _, l := range existingLines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			existingSet[trimmed] = true
		}
	}

	templateLines := strings.Split(string(templateData), "\n")
	var missing []string
	for _, l := range templateLines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !existingSet[trimmed] {
			missing = append(missing, l)
		}
	}

	if len(missing) == 0 {
		return existing, nil
	}

	result := string(existing)
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result += "\n"
	}
	result += "\n"
	for _, l := range missing {
		result += l + "\n"
	}

	return []byte(result), nil
}
