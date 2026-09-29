@echo off
cd /d "%USERPROFILE%\minecraft" || exit /b 1
java -Xmx4G -jar server.jar nogui
pause
