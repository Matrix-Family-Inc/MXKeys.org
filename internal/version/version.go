/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package version

const (
	Version = "0.2.0"
	Name    = "MXKeys"
)

func Full() string {
	return Name + "/" + Version
}
