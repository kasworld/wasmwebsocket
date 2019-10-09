# wasmwebsocket
golang wasm websocket 

goguelike2의 gopherjs webclient 를 go언에에서 (아직  EXPERIMENTAL 이라고 하지만) 정식으로 지원 하기 시작한 Webassembly 로 포팅하다가 나온 부산 물 입니다.

기본적으로
golang websocket server
서버가 정상 작동 하는 지는 테스트 하기 위한 golang websocket client
그리고 주 목적인 golang wasm web client
로 구성 되어 있습니다.

주의 할 점은 실행을 위해서 외부 패키지가 필요합니다.

golang websocket용 
github.com/gorilla/websocket

log 용
github.com/kasworld/log 

## genprotocol 을 사용한 protocol 코드 생성 
https://github.com/kasworld/genprotocol
을 사용해서 protocol의 기반 코드를 생성합니다. 

go get github.com/kasworld/genprotocol

으로 genprotocol 을 설치 하는 것이 필요합니다.  
