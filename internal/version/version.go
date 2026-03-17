/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

package version

const (
	Version = "0.1.0"
	Name    = "MXKeys"
)

func Full() string {
	return Name + "/" + Version
}
