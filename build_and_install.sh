#!/bin/bash

# MCL 빌드 및 설치 스크립트
# 이 스크립트는 mcl 프로젝트를 빌드하고 로컬에 설치합니다.

set -e  # 에러 발생 시 스크립트 중단

echo "🚀 MCL 빌드 및 설치를 시작합니다..."

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로그 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 현재 디렉토리가 프로젝트 루트인지 확인
if [ ! -f "go.mod" ]; then
    log_error "go.mod 파일을 찾을 수 없습니다. 프로젝트 루트 디렉토리에서 실행해주세요."
    exit 1
fi

# Go가 설치되어 있는지 확인
if ! command -v go &> /dev/null; then
    log_error "Go가 설치되어 있지 않습니다. https://golang.org/dl/ 에서 설치해주세요."
    exit 1
fi

log_info "Go 버전 확인 중..."
go version

# 의존성 정리
log_info "의존성을 정리하고 다운로드 중..."
go mod tidy
go mod download

# 빌드
log_info "MCL을 빌드 중..."
BINARY_NAME="mcl"
BUILD_DIR="./dist"

# 빌드 디렉토리 생성
mkdir -p "$BUILD_DIR"

# 현재 시간을 버전으로 사용
VERSION=$(date +"%Y%m%d_%H%M%S")

# 빌드 실행
log_info "바이너리 빌드 중..."
if go build -ldflags="-s -w -X main.mclVersion=v${VERSION}" -o "$BUILD_DIR/$BINARY_NAME" ./main.go; then
    log_success "빌드가 완료되었습니다!"
else
    log_error "빌드에 실패했습니다."
    exit 1
fi

# 설치 디렉토리 설정
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

# 기존 설치 확인
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    log_warning "기존 설치가 발견되었습니다. 백업을 생성합니다..."
    mv "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
fi

# 새 바이너리 설치
log_info "바이너리를 설치 중..."
cp "$BUILD_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# PATH에 추가
log_info "PATH 설정을 확인 중..."

# 사용자의 쉘 확인
SHELL_CONFIG=""
if [[ "$SHELL" == *"zsh"* ]]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    SHELL_CONFIG="$HOME/.bashrc"
else
    SHELL_CONFIG="$HOME/.profile"
fi

# PATH에 이미 추가되어 있는지 확인
if ! grep -q "$INSTALL_DIR" "$SHELL_CONFIG" 2>/dev/null; then
    log_info "PATH에 $INSTALL_DIR 추가 중..."
    echo "" >> "$SHELL_CONFIG"
    echo "# MCL 설치 경로" >> "$SHELL_CONFIG"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
    log_success "PATH가 $SHELL_CONFIG에 추가되었습니다."
else
    log_info "PATH가 이미 설정되어 있습니다."
fi

# 현재 세션에서도 PATH 업데이트
export PATH="$PATH:$INSTALL_DIR"

# 설치 확인
log_info "설치 확인 중..."
if command -v "$BINARY_NAME" &> /dev/null; then
    log_success "설치가 완료되었습니다!"
    echo ""
    echo "🎉 MCL이 성공적으로 설치되었습니다!"
    echo ""
    echo "📋 설치 정보:"
    echo "   - 바이너리 위치: $INSTALL_DIR/$BINARY_NAME"
    echo "   - 버전: v$VERSION"
    echo "   - PATH 설정: $SHELL_CONFIG"
    echo ""
    echo "💡 사용 방법:"
    echo "   - 새 터미널을 열거나 다음 명령어를 실행하세요:"
    echo "     source $SHELL_CONFIG"
    echo "   - 그 후 다음 명령어로 MCL을 사용할 수 있습니다:"
    echo "     $BINARY_NAME --help"
    echo ""
    echo "🔧 현재 세션에서 바로 사용하려면:"
    echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
else
    log_error "설치 확인에 실패했습니다."
    exit 1
fi

# 빌드 디렉토리 정리 (선택사항)
read -p "빌드 디렉토리를 정리하시겠습니까? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "빌드 디렉토리 정리 중..."
    rm -rf "$BUILD_DIR"
    log_success "빌드 디렉토리가 정리되었습니다."
fi

log_success "모든 작업이 완료되었습니다! 🎉" 