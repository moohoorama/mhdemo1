Unicode True
!include "MUI2.nsh"
!define MUI_ICON "app.ico"
!define MUI_UNICON "app.ico"

!ifndef GAME_EXE
!define GAME_EXE "../../bin/win/srpg.exe"
!endif
!ifndef INSTALLER_EXE
!define INSTALLER_EXE "../../bin/win/srpg-ver5-setup.exe"
!endif
Name "SRPG"
OutFile "${INSTALLER_EXE}"
InstallDir "$LOCALAPPDATA\Programs\SRPG"
InstallDirRegKey HKCU "Software\SRPG" "InstallDir"
RequestExecutionLevel user

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  File /oname=srpg.exe "${GAME_EXE}"
  SetOutPath "$INSTDIR\assets"
  File /r /x ".DS_Store" "../../assets/*"
  SetOutPath "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  CreateDirectory "$SMPROGRAMS\SRPG"
  CreateShortcut "$SMPROGRAMS\SRPG\SRPG.lnk" "$INSTDIR\srpg.exe" "" "$INSTDIR\srpg.exe"
  CreateShortcut "$SMPROGRAMS\SRPG\Uninstall.lnk" "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "Software\SRPG" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG" "DisplayName" "SRPG"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG" "DisplayIcon" "$INSTDIR\srpg.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$SMPROGRAMS\SRPG\SRPG.lnk"
  Delete "$SMPROGRAMS\SRPG\Uninstall.lnk"
  RMDir "$SMPROGRAMS\SRPG"
  Delete "$INSTDIR\srpg.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir /r "$INSTDIR\assets"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\SRPG"
  DeleteRegKey HKCU "Software\SRPG"
SectionEnd
