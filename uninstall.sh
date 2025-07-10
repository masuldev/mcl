#!/bin/bash

# MCL 제거 스크립트
# 이 스크립트는 mcl을 시스템에서 제거합니다.

set -e  # 에러 발생 시 스크립트 중단

echo "🗑️  MCL 제거를 시작합니다..."

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

BINARY_NAME="mcl"
INSTALL_DIR="$HOME/.local/bin"
BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"

# 사용자의 쉘 확인
SHELL_CONFIG=""
if [[ "$SHELL" == *"zsh"* ]]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    SHELL_CONFIG="$HOME/.bashrc"
else
    SHELL_CONFIG="$HOME/.profile"
fi

# 바이너리 제거
if [ -f "$BINARY_PATH" ]; then
    log_info "바이너리를 제거 중..."
    rm -f "$BINARY_PATH"
    log_success "바이너리가 제거되었습니다."
else
    log_warning "설치된 바이너리를 찾을 수 없습니다: $BINARY_PATH"
fi

# 백업 파일들도 제거
BACKUP_FILES=$(find "$INSTALL_DIR" -name "${BINARY_NAME}.backup.*" 2>/dev/null || true)
if [ -n "$BACKUP_FILES" ]; then
    log_info "백업 파일들을 제거 중..."
    echo "$BACKUP_FILES" | xargs rm -f
    log_success "백업 파일들이 제거되었습니다."
fi

# PATH 설정에서 제거
if [ -f "$SHELL_CONFIG" ]; then
    log_info "PATH 설정에서 MCL 관련 설정을 제거 중..."
    
    # 임시 파일 생성
    TEMP_FILE=$(mktemp)
    
    # MCL 관련 라인을 제외하고 복사
    grep -v "MCL 설치 경로" "$SHELL_CONFIG" | grep -v "export PATH.*$INSTALL_DIR" > "$TEMP_FILE"
    
    # 원본 파일 백업
    cp "$SHELL_CONFIG" "${SHELL_CONFIG}.backup.$(date +%Y%m%d_%H%M%S)"
    
    # 임시 파일을 원본으로 이동
    mv "$TEMP_FILE" "$SHELL_CONFIG"
    
    log_success "PATH 설정이 정리되었습니다."
else
    log_warning "쉘 설정 파일을 찾을 수 없습니다: $SHELL_CONFIG"
fi

# 빌드 디렉토리 제거
BUILD_DIR="./dist"
if [ -d "$BUILD_DIR" ]; then
    log_info "빌드 디렉토리를 제거 중..."
    rm -rf "$BUILD_DIR"
    log_success "빌드 디렉토리가 제거되었습니다."
fi

# 설치 디렉토리가 비어있으면 제거
if [ -d "$INSTALL_DIR" ] && [ -z "$(ls -A "$INSTALL_DIR" 2>/dev/null)" ]; then
    log_info "빈 설치 디렉토리를 제거 중..."
    rmdir "$INSTALL_DIR"
    log_success "빈 설치 디렉토리가 제거되었습니다."
fi

log_success "MCL 제거가 완료되었습니다! 🎉"
echo ""
echo "📋 제거된 항목:"
echo "   - 바이너리: $BINARY_PATH"
echo "   - PATH 설정: $SHELL_CONFIG에서 MCL 관련 설정 제거"
echo "   - 빌드 디렉토리: $BUILD_DIR"
echo ""
echo "💡 참고사항:"
echo "   - 새 터미널을 열거나 다음 명령어를 실행하여 변경사항을 적용하세요:"
echo "     source $SHELL_CONFIG"
echo "   - 쉘 설정 백업이 생성되었습니다: ${SHELL_CONFIG}.backup.*" 