---
title: "Hello World가 Console에 출력되기 까지"
date: 2026-06-27
slug: "hello-world-1"
description: "이 글은 Go 언어에서 fmt.Println이 내부적으로 fmt.Fprintln(os.Stdout, …)를 호출해 표준 출력으로 문자열을 전달하는 과정을 설명하고, 표준 스트림과 파일 디스크립터 개념을 통해 입출력 구조를 이해하도록 돕는다. 또한 fmt.Fprintln이 인자를 포맷하고 버퍼에 기록한 뒤 한 번에 io.Writer에 쓰는 구현 방식을 상세히 소개한다."
tags: ["Backend", "OS"]
categories: ["Go"]
cover: "cover.png"
draft: false
---

## 들어가며

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}
```

새로운 프로그래밍 언어를 배울 때 항상 등장 하는 국룰이 있습니다. 바로 콘솔에 `Hello World`를 찍어보는 것. 

최근에 사내에서 백엔드 기술스택을 Go로 전환하게 되면서 Go를 후다닥 배우고 나니 Go에서 Hello World는 어떻게 콘솔에 출력을 할 수 있는지 궁금해졌습니다.

Go에서는 표준 라이브러리인 `fmt` 패키지의 `Println` 함수를 사용하면 콘솔에 문자열을 출력할 수 있습니다.

## Println 내부 구현

```go
func Println(a ...any) (n int, err error) {
	return Fprintln(os.Stdout, a...)
}
```

Println의 구현을 살펴보면 내부에서 `Fprintln`을 호출하며 첫 번째 인자로 `os.Stdout`을 넘기고, 우리가 출력하라고 전달한 인자들(`a...`)도 그대로 함께 전달한다는 것을 확인 할 수 있습니다.

여기서 보이는 `os.Stdout`은 정확히 무엇일까요? os.Stdout을 이해하기 위해서는 프로그램이 어떻게 외부와 데이터를 주고 받는지(I/O)에 대해서 살펴봐야 합니다.

### 입력/출력 (I/O)

우리가 만드는 프로그램은 **입력과 출력(I/O)이 없으면** 그 존재 이유가 흐려지는 경우가 많습니다. 계산을 하든, 파일을 읽든, 네트워크로 통신하든, 결국 프로그램은 데이터를 받아서 내보내는 일을 합니다. 

리눅스에서는 프로세스가 이 I/O를 어떻게 다룰까요? 

> 리눅스에서는 **“거의 모든 것이 파일처럼 다뤄진다”** (everything is a file)

키보드 입력, 터미널 출력, 파일, 파이프, 소켓… 겉보기에는 다 다른 것들이지만, 유닉스 계열 운영체제는 이들을 가능한 한 파일처럼 취급하려고 합니다. 그리고 그 연결을 프로세스에게 제공하는게 바로 **파일 디스크립터(File Descriptor, FD)** 입니다.

중요한 점은, 프로세스의 읽기/쓰기가 결국 `read()`, `write()` 같은 **시스템 콜**로 통일된다는 것입니다. 어디에서 읽고 어디에 쓰는지는 `read(fd, ...)`, `write(fd, ...)`에서 **fd 번호가 무엇을 가리키느냐**로 결정됩니다. 프로세스가 시작될 때 운영체제는 기본으로 3개의 통로를 열어줍니다. 바로 **표준 스트림(standard streams)** 이죠.

### 표준 출력, 표준 입력, 표준 에러

운영체제는 프로세스가 시작될때 기본으로 3개의 **표준 스트림(standard streams)** 을 제공합니다. 

- **표준 입력 (stdin)**: 프로그램이 _읽는 쪽_ 기본 통로
  - 파일 디스크립터: **0**
  - Go에서는 `os.Stdin` (`*os.File`)
  - 보통은 터미널에서 사용자가 입력한 키보드 내용이 들어오지만, **리다이렉션/파이프**를 걸면 파일이나 다른 프로세스의 출력이 stdin으로 들어올 수 있습니다.
- **표준 출력 (stdout)**: 프로그램이 _정상 결과를 쓰는 쪽_ 기본 통로
  - 파일 디스크립터: **1**
  - Go에서는 `os.Stdout`
  - 터미널에 출력되는 로그/결과가 여기로 나가며, `>` 로 파일에 저장하거나 `|` 로 다른 프로그램에 넘길 수 있습니다.
- **표준 에러 (stderr)**: 프로그램이 _에러/진단 메시지를 쓰는 쪽_ 기본 통로
  - 파일 디스크립터: **2**
  - Go에서는 `os.Stderr`
  - stdout과 분리되는 이유는 **정상 출력(결과)** 과 **오류/디버그 출력**을 섞지 않기 위해서입니다. 예를 들어 stdout은 다음 단계 프로그램의 입력으로 파이프 연결하고(`|`), stderr은 터미널에 그대로 남기거나 별도 파일로 저장할 수 있습니다.

Go 표준 라이브러리에서 이 3개는 모두 파일처럼 취급됩니다.

- `os.Stdin`은 `io.Reader`
- `os.Stdout`, `os.Stderr`는 `io.Writer`

그래서 `fmt.Fprintln(os.Stdout, ...)` 처럼 “어디에 쓸지”를 `io.Writer`로 주입할 수 있고, 같은 함수로 파일/버퍼/네트워크 소켓 등에도 동일하게 출력할 수 있습니다.

앞에서 본 것처럼 `fmt.Println`은 내부에서 `fmt.Fprintln(os.Stdout, ...)`을 호출합니다. 즉, **Println은 기본 출력 대상으로 표준 출력(stdout)을 선택**하고, 그 표준 출력이 실제로는 터미널일 수도 있고 리다이렉션된 파일일 수도 있는 구조입니다.

```go
// These routines end in 'ln', do not take a format string,
// always add spaces between operands, and add a newline
// after the last operand.

// Fprintln formats using the default formats for its operands and writes to w.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Fprintln(w io.Writer, a ...any) (n int, err error) {
	p := newPrinter()
	p.doPrintln(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}

// doPrintln is like doPrint but always adds a space between arguments
// and a newline after the last argument.
func (p *pp) doPrintln(a []any) {
	for argNum, arg := range a {
		if argNum > 0 {
			p.buf.writeByte(' ')
		}
		p.printArg(arg, 'v')
	}
	p.buf.writeByte('\n')
}
```

코드 구현을 자세히 보면 `fmt.Fprintln`이 실제로 출력 문자열을 만드는 핵심 역할을 합니다. 내부의 `doPrintln` 로직에서 전달된 인자(`a ...any`)를 **순서대로 순회**하면서, **첫 번째 인자를 제외한 나머지 인자 앞에 공백(****`' '`****)을 하나씩 추가**해 출력합니다. 그리고 모든 인자 처리가 끝나면 **마지막에 개행 문자(****`'\n'`****)를 버퍼에 추가**해 한 줄 출력이 되도록 마무리합니다.

### 다음이야기

println의 내부 구현을 보면 버퍼를 통해 입력과 출력을 처리하는 것을 볼 수 있는데요. 다음 글에서는 왜 버퍼를 통해서 I/O를 처리하는지 알아보도록 하겠습니다.
