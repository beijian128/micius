::protoc.exe --go_out=. --proto_path=.;../../../ gamedef/*.proto
::protoc.exe --go_out=. --proto_path=.;../../../ opcode/*.proto
protoc.exe --go_out=. --proto_path=.;../../../ *.proto

@ECHO.
@ECHO Compile .proto To .go Done!
@PAUSE
