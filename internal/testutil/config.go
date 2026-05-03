/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

// Package testutil provides shared constants for integration tests and test setup.
package testutil

const (
	TestGrpcAddr = "localhost:9180"
	TestHTTPAddr = "http://localhost:8180"

	TestUser     = "testuser"
	TestEmail    = "testuser@test.co"
	TestPassword = "TestPassword123!"

	TestUser2     = "testuser2"
	TestEmail2    = "testuser2@test.co"
	TestPassword2 = "TestPassword456!"

	// sec-org: Private organisation for visibility/security tests.
	// testuser is Member, testuser2 is Owner + Coordinator of secret circle.
	SecOrg              = "sec-org"
	SecOrgPrivateCircle = "sec-org#private-circle"
	SecOrgSecretCircle  = "sec-org#secret-circle"

	// Projects seeded in sec-org
	SecOrgRootProject    = "Root Project"
	SecOrgPrivateProject = "Private Project"
	SecOrgSecretProject  = "Secret Project"

	// Project nameids (used by visibility tests)
	PublicProjectNameid  = "public-project"  // on test-org (Public)
	PrivateProjectNameid = "private-project" // on sec-org#private-circle (Private)
	SecretProjectNameid  = "secret-project"  // on sec-org#secret-circle (Secret)

	// ProjectColumn names (used by visibility tests)
	PublicColumnName  = "col-public"
	PrivateColumnName = "col-private"
	SecretColumnName  = "col-secret"

	// MinIO/S3 backend for /file/* tests (see docker-compose.test.yml).
	MinioAddr      = "localhost:9100"
	MinioAccessKey = "minioadmin"
	MinioSecretKey = "minioadmin"
	TestBucket     = "fractale-test"

	// Unique seed-comment markers used by file-attachment tests; the integration
	// suite resolves these to uids at runtime via Post.message lookup.
	FileTestPublicCommentByUser1  = "file-test: public comment by testuser"
	FileTestPublicCommentByUser2  = "file-test: public comment by testuser2"
	FileTestPrivateCommentByUser2 = "file-test: private-circle comment by testuser2"
)
