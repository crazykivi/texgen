package imageproc

type Pixel struct {
	R, G, B uint8
}

type Face struct {
	O, U, V [2]float64
}

type CubeGeom struct {
	Faces          map[string]Face
	Htop, Edge, Hw float64
}
