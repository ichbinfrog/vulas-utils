package promote

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/ichbinfrog/vulas-utils/internal/isolate"
)

func checkPrereqs() string {
	path, err := exec.LookPath("helm3")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("helm binary found on path %s", path)
	result, _ := exec.Command("helm3", "version", "--short").Output()
	log.Printf("current helm version %s", string(result))
	if !strings.Contains(string(result), "v3") {
		log.Fatal("requires helm 3")
	}

	return path
}

var (
	path = checkPrereqs()
)

func checkErr(result []byte) bool {
	return strings.Contains(strings.ToUpper(string(result)), "ERROR")
}

func helmLint(ChartDir string) {
	log.Println("Checking if new chart is valid")
	result, _ := exec.Command("helm3", "lint", ChartDir).Output()
	if checkErr(result) {
		log.Fatal(string(result))
	}
}

func helmList(context *Context) error {
	log.Println("Checking for given helm release existence")
	result, _ := exec.Command("helm3", "ls", "--short", "--namespace", context.CoreNamespace).Output()
	if checkErr(result) {
		log.Fatal(string(result))
	}
	if !strings.Contains(string(result), context.OldRelease) {
		log.Fatalf("Did not find given release %s in namespace %s", context.OldRelease, context.CoreNamespace)
	}
	log.Printf("Found charts %s", result)
	return nil
}

func HelmUpgrade(context *Context) error {
	if err := os.Chdir(context.ChartDir); err != nil {
		return err
	}

	helmLint(".")
	if listErr := helmList(context); listErr != nil {
		return listErr
	}

	statefulsetName := fmt.Sprintf("%s-database-slave", context.OldRelease)
	claimName := isolate.Isolate(&statefulsetName, &context.CoreNamespace)
	if claimName != nil {
		replaceFiles(context, *claimName)
		log.Printf("Installing new release %s", context.NewRelease)
		result, _ := exec.Command("helm3", "install", context.NewRelease, ".").Output()
		if checkErr(result) {
			log.Fatal(string(result))
		}
	} else {
		return fmt.Errorf("Encountered unknown error with fetching claimName")
	}

	return nil
}
