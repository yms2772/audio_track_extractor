package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"fyne.io/fyne/dialog"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	dlog "github.com/sqweek/dialog"
)

var (
	title      = "MKV Audio Track Extractor"
	fontInfo   string
	ffmpegInfo string
	runDir     string
)

func CryptoSHA256(file string) string {
	f, err := os.Open(file)
	ErrHandle(err)
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		ErrHandle(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func ErrHandle(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func RunAgain() {
	path, err := os.Executable()
	ErrHandle(err)

	err = exec.Command(path).Start()
	ErrHandle(err)

	os.Exit(1)
}

// Objects
func MainObject(w fyne.Window) fyne.CanvasObject {
	mkvEntry := widget.NewEntry()
	mkvEntry.SetText("'열기'를 눌러서 mkv 파일을 불러오세요.")
	mkvEntry.Disable()

	openFile := widget.NewButtonWithIcon("열기", theme.FolderOpenIcon(), func() {
		go func() {
			mkvFile, err := dlog.File().Filter("MKV 동영상", "mkv").Load()
			if err == nil {
				if len(mkvFile) != 0 {
					mkvEntry.SetText(mkvFile)
				}
			}
		}()
	})

	openFileLayout := widget.NewHBox(layout.NewSpacer(), openFile)

	form := &widget.Form{}
	form.Append("MKV 위치", mkvEntry)

	mkvEntry.OnChanged = func(mkvFile string) {
		if mkvFile == "'열기'를 눌러서 mkv 파일을 불러오세요." {
			return
		}

		fmt.Println("영상 불러오는 중...")

		_, err := os.Stat(mkvFile)

		if os.IsNotExist(err) || len(mkvFile) == 0 {
			dialog.ShowInformation(title, "잘못된 위치입니다.", w)

			return
		}

		prog := dialog.NewProgressInfinite(title, "영상 처리 중...", w)
		prog.Show()

		r, err := regexp.Compile(`Stream #\d+:\d+: Audio`)
		ErrHandle(err)

		fmt.Println("ffmpeg 실행 중...")

		cmd := exec.Command(ffmpegInfo, "-i", mkvFile)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		stderr, err := cmd.StderrPipe()
		ErrHandle(err)

		err = cmd.Start()
		ErrHandle(err)

		audioTrackNum := 0

		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			outputFFmpeg := scanner.Text()

			if r.MatchString(outputFFmpeg) {
				fmt.Println("오디오 트랙 +1")
				audioTrackNum++
			}
		}

		cmd.Wait()

		fmt.Printf("%d개의 오디오 트랙", audioTrackNum)

		wClose := 0
		moreInfoW := fyne.CurrentApp().NewWindow(title)

		mp4Entry := widget.NewEntry()
		mp4Entry.SetText("'열기'를 눌러서 저장할 곳을 선택하세요.")
		mp4Entry.Disable()

		openDir := widget.NewButtonWithIcon("열기", theme.FolderOpenIcon(), func() {
			go func() {
				saveDir, _ := dlog.Directory().Title("mp4 저장").Browse()
				mp4Entry.SetText(saveDir)
			}()
		})

		saveForm := &widget.Form{}
		saveForm.Append("오디오 트랙", widget.NewLabel(fmt.Sprintf("%d개", audioTrackNum)))
		saveForm.Append("MP4 저장", mp4Entry)

		saveDirLayout := widget.NewHBox(layout.NewSpacer(), openDir)

		mp4Entry.OnChanged = func(mp4Dir string) {
			fmt.Println(mp4Dir)

			_, err := os.Stat(mp4Dir)

			if os.IsNotExist(err) || len(mp4Dir) == 0 {
				dialog.ShowInformation(title, "잘못된 위치입니다.", w)

				return
			}

			saveProg := dialog.NewProgress(title, "mp4로 저장 중...", moreInfoW)

			for i := 1; i <= audioTrackNum; i++ {
				saveProg.SetValue(float64(i) / float64(audioTrackNum))

				cmd := exec.Command(ffmpegInfo, "-y", "-i", mkvFile, "-map", fmt.Sprintf("0:%d", i), fmt.Sprintf("%s/output_%d.mp4", mp4Dir, i))
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				_ = cmd.Run()

				fmt.Printf(fmt.Sprintf("저장: %s/output_%d.mp4\n", mp4Dir, i))
			}

			wClose = 1
		}

		saveDirObject := widget.NewVBox(
			layout.NewSpacer(),
			saveForm,
			saveDirLayout,
			layout.NewSpacer(),
		)

		moreInfoW.SetContent(saveDirObject)
		moreInfoW.Resize(fyne.NewSize(420, 180))
		moreInfoW.SetFixedSize(true)
		moreInfoW.CenterOnScreen()
		moreInfoW.Show()

		for wClose == 0 {
			time.Sleep(1 * time.Second)
		}

		moreInfoW.Close()
		prog.Hide()

		dialog.ShowInformation(title, fmt.Sprintf("오디오 트랙 %d개 추출 완료", audioTrackNum), w)
		mkvEntry.SetText("'열기'를 눌러서 mkv 파일을 불러오세요.")
	}

	return widget.NewVBox(
		layout.NewSpacer(),
		form,
		openFileLayout,
		layout.NewSpacer(),
	)
}

func main() {
	runDir, _ = filepath.Abs(filepath.Dir(os.Args[0]))

	fmt.Println(runDir)

	fontInfo = runDir + "/bin/AppleSDGothicNeoB.ttf"
	ffmpegInfo = runDir + "/bin/ffmpeg.exe"

	if _, err := os.Stat(fontInfo); err == nil {
		if CryptoSHA256(fontInfo) != "a652ea0a3c4bf8658845f044b5d6f40c39ecf03207e43f325c1451127528402b" {
			err := os.Remove(fontInfo)
			ErrHandle(err)

			RunAgain()
		}

		err = os.Setenv("FYNE_FONT", fontInfo)
		ErrHandle(err)
	}

	a := app.New()
	a.Settings().SetTheme(newCustomTheme(theme.TextFont()))

	w := a.NewWindow(title)

	w.SetOnClosed(func() {
		os.Exit(1)
	})

	w.CenterOnScreen()
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(420, 180))

	w.SetContent(MainObject(w))

	w.ShowAndRun()
}
