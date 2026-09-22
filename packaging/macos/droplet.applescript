-- Doble clic: abre Pasame. Soltar archivos encima: abre Pasame compartiéndolos
-- (si ya está abierto, el binario se los pasa a la instancia que corre).
on run
	launchPasame({})
end run

on open droppedItems
	launchPasame(droppedItems)
end open

on launchPasame(theItems)
	set bin to quoted form of (POSIX path of (path to me) & "Contents/Resources/pasame")
	set args to ""
	repeat with f in theItems
		set args to args & " " & quoted form of (POSIX path of f)
	end repeat
	do shell script bin & args & " > /dev/null 2>&1 &"
end launchPasame
