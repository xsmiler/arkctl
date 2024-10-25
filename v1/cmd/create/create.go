package create

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/koupleless/arkctl/v1/cmd/root"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//go:embed koupleless-ext-module-auto-convertor-0.0.1-SNAPSHOT.jar
var jarFile []byte

var createCmd = &cobra.Command{
	Use:   "create [flags]",
	Short: "转换为模块自动配置",
	Long: `create 命令用于自动模块化。
它会执行以下操作:
1. 修改 application.properties 文件
2. 创建 bootstrap.properties 文件
3. 修改 pom.xml 文件

使用方法:
  arkctl create -p <项目路径> -a <应用名称>

示例:
  arkctl create -p /path/to/project -a myapp`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("projectPath")
		applicationName, _ := cmd.Flags().GetString("applicationName")

		if err := runJavaProgram(projectPath, applicationName); err != nil {
			fmt.Fprintf(os.Stderr, "执行 create 命令失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func runJavaProgram(projectPath, applicationName string) error {
	tempDir, err := createTempJarFile()
	if err != nil {
		return fmt.Errorf("创建临时 JAR 文件失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	jarPath := filepath.Join(tempDir, "converter.jar")
	cmd := prepareJavaCommand(jarPath, projectPath)

	if err := executeJavaCommand(cmd, projectPath, applicationName); err != nil {
		return fmt.Errorf("执行 Java 命令失败: %w", err)
	}

	return nil
}

func createTempJarFile() (string, error) {
	tempDir, err := os.MkdirTemp("", "arkctl-jar")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	jarPath := filepath.Join(tempDir, "converter.jar")
	if err := os.WriteFile(jarPath, jarFile, 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("写入 jar 文件失败: %w", err)
	}

	return tempDir, nil
}

func prepareJavaCommand(jarPath, projectPath string) *exec.Cmd {
	cmd := exec.Command("java", "-Dfile.encoding=UTF-8", "-jar", jarPath)
	cmd.Dir = projectPath
	return cmd
}

func executeJavaCommand(cmd *exec.Cmd, projectPath, applicationName string) error {
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建标准输入管道失败: %w", err)
	}

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &outBuffer

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 Java 程序出错: %w", err)
	}

	if err := writeInputToJavaProgram(stdinPipe, projectPath, applicationName); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		printGBKString(outBuffer.Bytes())
		return fmt.Errorf("Java 程序运行出错: %w", err)
	}

	printGBKString(outBuffer.Bytes())
	return nil
}

func writeInputToJavaProgram(stdinPipe io.WriteCloser, projectPath, applicationName string) error {
	if _, err := io.WriteString(stdinPipe, projectPath+"\n"); err != nil {
		return fmt.Errorf("写入项目路径出错: %w", err)
	}

	if _, err := io.WriteString(stdinPipe, applicationName+"\n"); err != nil {
		return fmt.Errorf("写入应用名称出错: %w", err)
	}

	if err := stdinPipe.Close(); err != nil {
		return fmt.Errorf("关闭标准输入管道出错: %w", err)
	}

	return nil
}

func printGBKString(gbkBytes []byte) {
	utf8Reader := transform.NewReader(bytes.NewReader(gbkBytes), simplifiedchinese.GBK.NewDecoder())
	utf8Bytes, err := io.ReadAll(utf8Reader)
	if err != nil {
		log.Printf("转换编码出错: %v", err)
		return
	}
	fmt.Printf("Java 程序输出:\n%s\n", string(utf8Bytes))
}

func init() {
	root.RootCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("projectPath", "p", "", "项目路径 (必填)")
	createCmd.Flags().StringP("applicationName", "a", "", "应用名称 (必填)")

	createCmd.MarkFlagRequired("projectPath")
	createCmd.MarkFlagRequired("applicationName")
}
