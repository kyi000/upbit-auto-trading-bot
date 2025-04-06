# 업비트 자동 트레이딩 봇 설치 및 실행 가이드

이 가이드는 업비트 자동 트레이딩 봇을 구축하고 실행하기 위한 모든 단계를 설명합니다. 윈도우 환경에서 WSL2 Ubuntu를 활용하는 방법을 기준으로 작성되었습니다.

## 1. 시스템 요구사항

- Windows 10 이상 (Windows 11 권장)
- WSL2 Ubuntu 24.04
- 최소 8GB RAM
- 최소 50GB 디스크 공간
- 업비트 거래소 API 키 (액세스 키 & 시크릿 키)

## 2. 필수 소프트웨어 설치

### Windows 환경에서 설치

```powershell
# WSL2 설치 (PowerShell 관리자 권한으로 실행)
wsl --install -d Ubuntu-24.04

# Node.js 설치 (https://nodejs.org/에서 LTS 버전 다운로드 및 설치)

# Git 설치 (https://git-scm.com/download/win 에서 다운로드 및 설치)

# Visual Studio Code 설치 (선택사항) (https://code.visualstudio.com/download)
```

### WSL2 Ubuntu 환경에서 설치

```bash
# WSL2 Ubuntu 터미널에서 실행

# 시스템 업데이트
sudo apt update
sudo apt upgrade -y

# Go 설치
sudo apt install -y golang-go

# PostgreSQL 설치
sudo apt install -y postgresql postgresql-contrib

# 기타 필요한 도구 설치
sudo apt install -y build-essential git curl wget
```

## 3. PostgreSQL 데이터베이스 설정

```bash
# PostgreSQL 서비스 시작
sudo service postgresql start

# PostgreSQL 초기 설정
sudo -u postgres psql -c "CREATE DATABASE upbit_trader;"
sudo -u postgres psql -c "CREATE USER upbit_user WITH ENCRYPTED PASSWORD 'your_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE upbit_trader TO upbit_user;"
```

## 4. 프로젝트 클론 및 설정

```bash
# 프로젝트 클론
git clone https://github.com/kyi000/upbit-auto-trading-bot.git
cd upbit-auto-trading-bot

# 설정 디렉토리 생성
mkdir -p ~/.upbit-trader

# 예제 설정 파일 복사
cp configs/config.example.yaml ~/.upbit-trader/config.yaml

# 설정 파일 편집 (API 키 등 설정)
nano ~/.upbit-trader/config.yaml
```

## 5. Go 백엔드 빌드

```bash
# WSL2 Ubuntu 환경에서 실행

# 프로젝트 디렉토리로 이동
cd ~/upbit-auto-trading-bot

# Go 의존성 설치
go mod download
go mod tidy

# 백엔드 빌드
mkdir -p bot
go build -o bot/upbit-trading-bot cmd/bot/main.go

# 빌드된 바이너리에 실행 권한 부여
chmod +x bot/upbit-trading-bot
```

## 6. React 프론트엔드 빌드

```bash
# Windows 환경에서 실행 (프로젝트 루트 디렉토리)

# 의존성 설치
cd web
npm install

# 프론트엔드 빌드
npm run build

# 빌드 결과물은 web/build 디렉토리에 생성됩니다
```

## 7. Electron 애플리케이션 빌드

```bash
# Windows 환경에서 실행 (프로젝트 루트 디렉토리)

# Electron 디렉토리로 이동
cd electron

# 의존성 설치
npm install

# React 빌드 결과물 복사
mkdir -p build
cp -r ../web/build/* build/

# 개발 모드 실행 (테스트용)
npm start

# 배포용 빌드 (인스톨러 생성)
npm run build
```

배포용 빌드가 완료되면 `electron/dist` 디렉토리에 Windows 인스톨러(.exe)가 생성됩니다.

## 8. 봇 실행 방법

### 방법 1: 직접 실행 (개발 환경)

```bash
# WSL2 Ubuntu 환경에서 백엔드 실행
cd ~/upbit-auto-trading-bot
./bot/upbit-trading-bot -config ~/.upbit-trader/config.yaml

# 별도 터미널에서 프론트엔드 개발 서버 실행 (Windows)
cd web
npm start
```

### 방법 2: 실행 스크립트 사용

```bash
# 실행 스크립트 생성
cat > start.sh << 'EOF'
#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CONFIG_DIR="$HOME/.upbit-trader"

# 데이터베이스 서비스 시작
sudo service postgresql start

# 로그 디렉토리 확인
mkdir -p logs

# 봇 실행
cd "$DIR"
./bot/upbit-trading-bot -config "$CONFIG_DIR/config.yaml"
EOF

# 스크립트에 실행 권한 부여
chmod +x start.sh

# 스크립트 실행
./start.sh
```

### 방법 3: Electron 애플리케이션 실행 (배포 환경)

1. 생성된 인스톨러(`electron/dist/Upbit Trading Bot Setup x.x.x.exe`)를 실행하여 애플리케이션을 설치합니다.
2. 설치된 애플리케이션을 실행합니다.
3. 첫 실행 시 API 키 설정 화면이 나타납니다. 업비트 API 키를 입력합니다.

## 9. 문제 해결

- **데이터베이스 연결 오류**: PostgreSQL 서비스가 실행 중인지 확인
  ```bash
  sudo service postgresql status
  sudo service postgresql start
  ```

- **API 키 오류**: config.yaml 파일에서 API 키가 올바르게 설정되었는지 확인
  ```bash
  nano ~/.upbit-trader/config.yaml
  ```

- **로그 확인**: 오류 발생 시 로그 파일 확인
  ```bash
  cat logs/upbit-trader.log
  ```

- **WSL 환경 재시작**:
  ```bash
  # Windows PowerShell에서
  wsl --shutdown
  # 다시 WSL 터미널 열기
  ```
