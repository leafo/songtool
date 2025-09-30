# songtool

Experiments working with rockband file format and conversion to other formats


```
Usage of ./songtool:
  -export-gm
    	Export drums, vocals, and bass to single General MIDI file
  -export-gm-bass
    	Export pro bass to General MIDI file
  -export-gm-drums
    	Export drum patterns to General MIDI file
  -export-gm-vocals
    	Export vocal melody to General MIDI file
  -export-tonelib-song
    	Create complete ToneLib .song file (ZIP archive)
  -export-tonelib-xml
    	Export to ToneLib the_song.dat XML format
  -extract-file string
    	Extract and print contents of specified file from SNG package to stdout
  -filter-track string
    	Filter to show only tracks whose name contains this string (case-insensitive)
  -json
    	Output information as JSON (supported with: default analysis, --timeline)
  -timeline
    	Print beat timeline from BEAT track
```


## TODO

[] quantization on tonelib song export is fixed to 8th notes right now, shold be much more accurate

## Resources

* <https://therogerland.tumblr.com/proguide>
* <http://docs.c3universe.com/rbndocs/index.php>
