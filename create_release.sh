#!/bin/bash

# GitHub Release 생성 스크립트
# 이 스크립트는 새로운 버전의 릴리스를 생성합니다.

set -e  # 에러 발생 시 스크립트 중단

echo "🚀 GitHub Release 생성을 시작합니다..."

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

# 사용법 출력
usage() {
    echo "사용법: $0 <버전> [릴리스 노트 파일]"
    echo ""
    echo "예시:"
    echo "  $0 v1.1.2"
    echo "  $0 v1.1.2 releases/v1.1.2.md"
    echo ""
    echo "옵션:"
    echo "  <버전>        릴리스할 버전 (예: v1.1.2)"
    echo "  [릴리스 노트]  릴리스 노트 파일 경로 (기본값: releases/<버전>.md)"
}

# 인수 확인
if [ $# -lt 1 ]; then
    log_error "버전을 지정해주세요."
    usage
    exit 1
fi

VERSION=$1
RELEASE_NOTES_FILE=${2:-"releases/${VERSION}.md"}

# 버전 형식 확인
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    log_error "올바른 버전 형식을 사용해주세요. (예: v1.1.2)"
    exit 1
fi

log_info "릴리스 버전: $VERSION"
log_info "릴리스 노트 파일: $RELEASE_NOTES_FILE"

# 릴리스 노트 파일 확인
if [ ! -f "$RELEASE_NOTES_FILE" ]; then
    log_error "릴리스 노트 파일을 찾을 수 없습니다: $RELEASE_NOTES_FILE"
    log_info "releases 디렉토리의 파일 목록:"
    ls -la releases/ 2>/dev/null || echo "releases 디렉토리가 없습니다."
    exit 1
fi

# Git 상태 확인
if ! git status --porcelain | grep -q .; then
    log_info "작업 디렉토리가 깨끗합니다."
else
    log_warning "커밋되지 않은 변경사항이 있습니다."
    read -p "계속하시겠습니까? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "릴리스 생성을 취소했습니다."
        exit 0
    fi
fi

# 현재 브랜치 확인
CURRENT_BRANCH=$(git branch --show-current)
log_info "현재 브랜치: $CURRENT_BRANCH"

# 태그가 이미 존재하는지 확인
if git tag -l | grep -q "^$VERSION$"; then
    log_error "태그 $VERSION이 이미 존재합니다."
    exit 1
fi

# 릴리스 노트 내용 읽기
RELEASE_NOTES=$(cat "$RELEASE_NOTES_FILE")

# Git 태그 생성
log_info "GPG 서명된 Git 태그를 생성 중..."
git tag -s "$VERSION" -m "Release $VERSION"

# 태그 푸시
log_info "태그를 원격 저장소에 푸시 중..."
git push origin "$VERSION"

# GitHub CLI가 설치되어 있는지 확인
if command -v gh &> /dev/null; then
    log_info "GitHub CLI를 사용하여 릴리스를 생성 중..."
    
    # 빌드 파일 생성
    log_info "릴리스용 바이너리를 빌드 중..."
    BUILD_DIR="./dist"
    BINARY_NAME="mcl"
    mkdir -p "$BUILD_DIR"

    # 의존성 정리 및 빌드
    go mod tidy
    go mod download

    if go build -ldflags="-s -w -X main.mclVersion=$VERSION" -o "$BUILD_DIR/$BINARY_NAME" ./main.go; then
        log_success "바이너리 빌드가 완료되었습니다!"
    else
        log_error "바이너리 빌드에 실패했습니다."
        exit 1
    fi

    # GitHub 릴리스 생성 (빌드 파일 포함)
    if echo "$RELEASE_NOTES" | gh release create "$VERSION" "$BUILD_DIR/$BINARY_NAME" --title "MCL $VERSION" --notes-file -; then
        log_success "GitHub 릴리스가 성공적으로 생성되었습니다!"
    else
        log_error "GitHub 릴리스 생성에 실패했습니다."
        exit 1
    fi
else
    log_warning "GitHub CLI가 설치되어 있지 않습니다."
    
    # 빌드 파일 생성 (GitHub CLI가 없어도)
    log_info "릴리스용 바이너리를 빌드 중..."
    BUILD_DIR="./dist"
    BINARY_NAME="mcl"
    mkdir -p "$BUILD_DIR"
    
    # 의존성 정리 및 빌드
    go mod tidy
    go mod download
    
    if go build -ldflags="-s -w -X main.mclVersion=$VERSION" -o "$BUILD_DIR/$BINARY_NAME" ./main.go; then
        log_success "바이너리 빌드가 완료되었습니다!"
        echo ""
        echo "📦 빌드된 파일:"
        echo "   - $BUILD_DIR/$BINARY_NAME"
        echo ""
    else
        log_error "바이너리 빌드에 실패했습니다."
        exit 1
    fi
    
    log_info "GitHub 웹사이트에서 수동으로 릴리스를 생성해주세요:"
    echo ""
    echo "1. https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\).*/\1/')/releases/new"
    echo "2. 태그: $VERSION"
    echo "3. 제목: MCL $VERSION"
    echo "4. 릴리스 노트 내용:"
    echo ""
    echo "$RELEASE_NOTES"
    echo ""
    echo "5. 바이너리 파일 업로드: $BUILD_DIR/$BINARY_NAME"
    echo ""
fi

# 로컬 빌드 스크립트 업데이트 (선택사항)
read -p "build_and_install.sh 스크립트의 버전을 업데이트하시겠습니까? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "build_and_install.sh 스크립트를 업데이트 중..."
    
    # 임시 파일 생성
    TEMP_FILE=$(mktemp)
    
    # 버전 정보를 업데이트
    sed "s/VERSION=\$(date +\"%Y%m%d_%H%M%S\")/VERSION=\"$VERSION\"/" build_and_install.sh > "$TEMP_FILE"
    
    # 원본 파일 백업
    cp build_and_install.sh "build_and_install.sh.backup.$(date +%Y%m%d_%H%M%S)"
    
    # 임시 파일을 원본으로 이동
    mv "$TEMP_FILE" build_and_install.sh
    
    log_success "build_and_install.sh 스크립트가 업데이트되었습니다."
fi

# 빌드 디렉토리 정리 (선택사항)
read -p "빌드 디렉토리를 정리하시겠습니까? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "빌드 디렉토리 정리 중..."
    rm -rf "$BUILD_DIR"
    log_success "빌드 디렉토리가 정리되었습니다."
fi

log_success "릴리스 생성이 완료되었습니다! 🎉"
echo ""
echo "📋 생성된 항목:"
echo "   - Git 태그: $VERSION"
echo "   - 릴리스 노트: $RELEASE_NOTES_FILE"
echo "   - GitHub 릴리스: MCL $VERSION"
echo "   - 바이너리 파일: $BUILD_DIR/$BINARY_NAME"
echo ""
echo "💡 다음 단계:"
echo "   1. GitHub에서 릴리스를 확인하세요"
echo "   2. 바이너리 파일이 자동으로 업로드되었습니다"
echo "   3. 릴리스 노트를 검토하고 필요시 수정하세요" 