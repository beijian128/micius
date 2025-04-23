@ECHO OFF

protoc --go_out=. -I=.;../../ cmsg/*.proto
@IF %ERRORLEVEL% NEQ 0 PAUSE

protoc --js_out=import_style=commonjs,binary:. cmsg/*.proto

protoc --go_out=. -I=.;../../ smsg/*.proto
@IF %ERRORLEVEL% NEQ 0 PAUSE



ECHO.
ECHO Compile .proto To .go Done!
@IF %ERRORLEVEL% NEQ 0 PAUSE