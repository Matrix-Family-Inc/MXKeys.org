/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"encoding/hex"
	"fmt"
)

func (tl *TransparencyLog) rebuildMerkleTree(ctx context.Context) error {
	query := fmt.Sprintf(`SELECT entry_hash FROM %s ORDER BY id ASC`, tl.tableName)
	rows, err := tl.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var entryHash string
		if err := rows.Scan(&entryHash); err != nil {
			return err
		}
		if err := tl.addMerkleHash(entryHash); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (tl *TransparencyLog) addMerkleHash(entryHash string) error {
	hashBytes, err := hex.DecodeString(entryHash)
	if err != nil {
		return fmt.Errorf("failed to decode transparency entry hash: %w", err)
	}
	tl.merkleTree.AddHash(hashBytes)
	return nil
}
