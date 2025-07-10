# MCL (My CLI)

AWS 서비스와 인증 서비스를 선택할 수 있는 인터랙티브 CLI 도구입니다.

## 기능

- EC2 인스턴스 관리
- RDS 인스턴스 관리  
- 볼륨 관리 (확장, 체크)
- AWS SSM을 통한 명령 실행

## 설치 방법

### 자동 설치 (권장)

프로젝트 루트 디렉토리에서 다음 명령어를 실행하세요:

```bash
./build_and_install.sh
```

이 스크립트는 다음 작업을 자동으로 수행합니다:

1. Go 의존성 다운로드 및 정리
2. MCL 바이너리 빌드
3. `~/.local/bin` 디렉토리에 설치
4. PATH에 자동 추가
5. 설치 확인

### 수동 설치

1. 의존성 다운로드:
```bash
go mod tidy
go mod download
```

2. 빌드:
```bash
go build -o mcl main.go
```

3. 설치:
```bash
mkdir -p ~/.local/bin
cp mcl ~/.local/bin/
chmod +x ~/.local/bin/mcl
```

4. PATH에 추가 (zsh 사용 시):
```bash
echo 'export PATH="$PATH:~/.local/bin"' >> ~/.zshrc
source ~/.zshrc
```

## 사용법

### 기본 사용법

```bash
mcl --help
```

### EC2 인스턴스 관리

```bash
# 모든 EC2 인스턴스 목록 보기
mcl ec2

# 특정 인스턴스 선택
mcl ec2 --target i-1234567890abcdef0

# 특정 그룹의 인스턴스 선택
mcl ec2 --group production
```

### 볼륨 관리

```bash
# 볼륨 사용량 체크 (기본값)
mcl volume

# 볼륨 확장
mcl volume --function expand

# 사용량 임계값 설정 (기본값: 80%)
mcl volume --threshold 70

# 확장 비율 설정 (기본값: 30%)
mcl volume --increment 50
```

### AWS 설정

```bash
# 특정 프로필 사용
mcl --profile my-profile

# 특정 리전 사용
mcl --region us-west-2

# 프로필과 리전 동시 사용
mcl --profile my-profile --region us-west-2
```

## 제거

MCL을 제거하려면 다음 명령어를 실행하세요:

```bash
./uninstall.sh
```

## 환경 변수

다음 환경 변수를 설정할 수 있습니다:

- `AWS_PROFILE`: 사용할 AWS 프로필
- `AWS_REGION`: 사용할 AWS 리전
- `AWS_ACCESS_KEY_ID`: AWS 액세스 키
- `AWS_SECRET_ACCESS_KEY`: AWS 시크릿 키
- `AWS_SESSION_TOKEN`: AWS 세션 토큰

## 개발

### 의존성

- Go 1.18 이상
- AWS CLI (선택사항)

### 빌드

```bash
go build -o mcl main.go
```

### 테스트

```bash
go test ./...
```

## 문제 해결

### PATH 문제

설치 후 `mcl` 명령어를 찾을 수 없는 경우:

```bash
export PATH="$PATH:~/.local/bin"
```

### 권한 문제

바이너리 실행 권한이 없는 경우:

```bash
chmod +x ~/.local/bin/mcl
```

### AWS 인증 문제

AWS 자격 증명이 올바르게 설정되어 있는지 확인하세요:

```bash
aws configure list
```

## 라이선스

이 프로젝트는 개인 사용을 위한 private 도구입니다.

## 기여

이 프로젝트는 개인 사용을 위한 도구이므로 외부 기여는 받지 않습니다. 