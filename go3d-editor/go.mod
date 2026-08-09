module go3deditor

go 1.26.5

require (
	go2dgame v0.0.0
	go3d v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.44.0
)

require (
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace (
	go2dgame => ../go2dgame
	go3d => ../go3d
)
