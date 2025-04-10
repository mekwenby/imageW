package main

import (
	"flag"
	"fmt"
	"github.com/chai2010/webp"
	"image"
	_ "image/jpeg" // 用于读取jpeg格式
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type File struct {
	Root string
	Name string
}

func getDirAllImageFiles(directory *string) ([]File, error) {
	var fileList []File

	supportedExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".bmp": true}

	err := filepath.Walk(*directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // 若访问文件出错，直接返回错误
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if supportedExtensions[ext] {
				fileList = append(fileList, File{
					Root: filepath.Dir(path),
					Name: info.Name(),
				})
			}
		}
		return nil
	})

	return fileList, err
}

// 打印帮助
func printHelp() {
	fmt.Println("使用方法:")
	fmt.Println("  myapp [选项]")
	fmt.Println("\n选项:")
	flag.PrintDefaults()
}

// 判断是否为目录
func isDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err // 可能是路径不存在或无权限访问
	}
	return info.IsDir(), nil
}

// 转换webp
func establishWebp(input *string, output *string, quality *int) {
	file, err := os.Open(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// 解码jpeg图像
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding image: %v\n", err)
		return
	}

	// 将image转换为webp并保存
	outputFile, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	// 使用默认质量选项（80）进行编码
	options := &webp.Options{Lossless: false, Quality: float32(*quality)}
	if err := webp.Encode(outputFile, img, options); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding to webp: %v\n", err)
		return
	}

	fmt.Printf("Image successfully converted to webp and saved as %v; quality %v\n", *output, *quality)

}

func main() {

	input := flag.String("i", "input.jpg", "输入文件")
	quality := flag.Int("q", 90, "设置质量")
	thread := flag.Int("t", 0, "设置线程数")
	help := flag.Bool("help", false, "显示帮助信息")

	// 解析命令行参数
	flag.Parse()
	// 获取逻辑处理器的数量
	numCPU := *thread
	if *thread == 0 {
		numCPU = runtime.NumCPU()
	}
	// 创建一个 Channel 来限制并发数量
	sem := make(chan struct{}, numCPU)
	var wg sync.WaitGroup
	// 如果用户输入 `-help`，则打印帮助信息并退出
	if *help {
		printHelp()
		os.Exit(0)
	}

	// 判断是否有输入文件
	isDir, err := isDirectory(*input)
	if err != nil {
		fmt.Println("错误:", err)
		return
	}
	if isDir {
		files, err := getDirAllImageFiles(input)
		indexNum := len(files)
		if err != nil {
			fmt.Println("读取目录错误:", err)
			return
		}
		for index, file := range files {
			filePath := fmt.Sprintf("%s/%s", file.Root, file.Name)
			output := fmt.Sprintf("%s/%s.webp", file.Root, file.Name)
			wg.Add(1)
			go func(index int, filePath, output string) {
				defer wg.Done()

				// 获取信号量，确保并发数不超过 numCPU
				sem <- struct{}{}
				defer func() { <-sem }() // 完成后释放信号量

				// 执行文件转换操作
				fmt.Printf("%d/%d\n正在转换文件: %s\n", index, indexNum, filePath)
				establishWebp(&filePath, &output, quality)
			}(index, filePath, output)
		}
	} else {
		output := fmt.Sprintf("%s.webp", *input)
		establishWebp(input, &output, quality)
	}
	wg.Wait()
}
