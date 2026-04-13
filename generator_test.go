package swift

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRejectsServerOption(t *testing.T) {
	schema := `
webrpc = v1

name = Demo
version = v1.0.0
basepath = /rpc

service Demo
  - Ping()
`

	errOutput := generateSwiftErr(t, schema, "-server")
	requireContains(t, errOutput, `-server="" is not supported target option`)
}

func TestSuccinctGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Demo
version = v1.0.0
basepath = /rpc

struct FlattenRequest
  - name: string
  - amount: uint64

struct FlattenResponse
  - id: uint64
  - counter: uint64

service Demo
  - Flatten(FlattenRequest) => (FlattenResponse)
  - SendPair(first: string, second: string) => (accepted: bool, note: string)
`

	output := generateSwift(t, schema)

	requireRegexp(t, `public func flatten\(_ request:\s*FlattenRequest\)\s*async throws ->\s*FlattenResponse`, output)
	requireRegexp(t, `public static func encodeRequest\(\s*_ request:\s*FlattenRequest,`, output)
	requireRegexp(t, `public static func decodeResponse\(\s*_ data:\s*Data,`, output)

	flattenBlock := regexp.MustCompile(`public enum Flatten \{(?s:.*?)\n    \}\n\n    public enum SendPair`).FindString(output)
	if flattenBlock == "" {
		t.Fatalf("expected to find Flatten method block\n\n%s", output)
	}
	requireNotContains(t, flattenBlock, "struct Request")
	requireNotContains(t, flattenBlock, "struct Response")

	requireRegexp(t, `public func sendPair\(_ request:\s*DemoAPI\.SendPair\.Request\)\s*async throws ->\s*DemoAPI\.SendPair\.Response`, output)
	requireRegexp(t, `public enum SendPair \{(?s:.*?)public struct Request`, output)
	requireRegexp(t, `public enum SendPair \{(?s:.*?)public struct Response`, output)
}

func TestSchemaAwareServiceNaming(t *testing.T) {
	waasSchema := `
webrpc = v1

name = waas
version = v1.0.0
basepath = /rpc

service Wallet
  - Ping()
`

	waasOutput := generateSwift(t, waasSchema)
	requireContains(t, waasOutput, "public enum WaasWalletAPI")
	requireContains(t, waasOutput, "public struct WaasWalletClient")
	requireContains(t, waasOutput, `public static let basePath = "/rpc/Wallet"`)

	testSchema := `
webrpc = v1

name = Test
version = v1.0.0
basepath = /rpc

service TestAPI
  - Ping()
`

	testOutput := generateSwift(t, testSchema)
	requireContains(t, testOutput, "public enum TestAPI")
	requireContains(t, testOutput, "public struct TestAPIClient")
	requireNotContains(t, testOutput, "TestTestAPI")
}

func TestAcronymHeavyNamingGeneralizes(t *testing.T) {
	schema := `
webrpc = v1

name = Acronyms
version = v1.0.0
basepath = /rpc

enum AuthMode: string
  - IDToken
  - AuthCodePKCE
  - OAuthDevice
  - OIDC

struct TokenInfo
  - oauthToken: string
  - apiURL: string
  - htmlBody: string
  - jwtValue: string

struct ExchangeResponse
  - mode: AuthMode

service OAuthAPI
  - ExchangeOAuthIDToken(TokenInfo) => (ExchangeResponse)

error 7001 OAuthError "oauth error"
`

	output := generateSwift(t, schema)

	requireContains(t, output, "case idToken")
	requireContains(t, output, "case authCodePkce")
	requireContains(t, output, "case oauthDevice")
	requireContains(t, output, "case oidc")
	requireContains(t, output, "public let oauthToken: String")
	requireContains(t, output, "public let apiUrl: String")
	requireContains(t, output, "public let htmlBody: String")
	requireContains(t, output, "public let jwtValue: String")
	requireContains(t, output, "public func exchangeOauthIdToken(")
	requireContains(t, output, "case oauthError")
}

func TestFieldNamesEscapeSwiftKeywords(t *testing.T) {
	schema := `
webrpc = v1

name = Keywords
version = v1.0.0
basepath = /rpc

struct Demo
  - await: string
  - precedencegroup: string

service Keywords
  - Echo(Demo) => (Demo)
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public let `await`: String")
	requireContains(t, output, "public let `precedencegroup`: String")
	requireContains(t, output, "case `await` = \"await\"")
	requireContains(t, output, "case `precedencegroup` = \"precedencegroup\"")
}

func TestNullOptionalGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Nulls
version = v1.0.0
basepath = /rpc

struct MaybeNull
  - value?: null

service Nulls
  - Echo(value?: null) => (value?: null)
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public let value: WebRPCNull?")
	requireContains(t, output, "public struct MaybeNull: Codable, Sendable")
	requireContains(t, output, "public func echo(")
}

func TestEnumMapGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Maps
version = v1.0.0
basepath = /rpc

enum Flavor: string
  - Vanilla
  - Chocolate

struct Basket
  - counts: map<Flavor, uint32>

service Maps
  - Echo(Basket) => (Basket)
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public enum Flavor: Codable, Hashable, Sendable, WebRPCEnumKey")
	requireContains(t, output, "public init(wireValue: String)")
	requireContains(t, output, "public let counts: WebRPCEnumMap<Flavor, UInt32>")
}

func TestServiceNameCollisionGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Collide
version = v1.0.0
basepath = /rpc

struct WalletAPI
  - id: uint64

struct WalletServiceAPI
  - id: uint64

struct WalletWebRPCAPI
  - id: uint64

struct WalletClient
  - id: uint64

struct WalletServiceClient
  - id: uint64

struct WalletWebRPCClient
  - id: uint64

struct CollideWalletAPI
  - id: uint64

struct CollideWalletServiceAPI
  - id: uint64

struct CollideWalletWebRPCAPI
  - id: uint64

struct CollideWalletClient
  - id: uint64

struct CollideWalletServiceClient
  - id: uint64

struct CollideWalletWebRPCClient
  - id: uint64

service Wallet
  - Ping()
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public enum CollideWalletGeneratedRPCAPI")
	requireContains(t, output, "public struct CollideWalletGeneratedRPCClient: Sendable")
}

func TestMultiServiceGenerationSeparatesTopLevelDeclarations(t *testing.T) {
	schema := `
webrpc = v1

name = Foo
version = v1.0.0
basepath = /rpc

service Bar
  - Ping()

service Baz
  - Pong()
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public enum FooBarAPI")
	requireContains(t, output, "public struct FooBarClient: Sendable")
	requireContains(t, output, "public enum FooBazAPI")
	requireContains(t, output, "public struct FooBazClient: Sendable")
	requireNotContains(t, output, "}public enum ")
}

func TestCrossServiceSchemaAwareNameCollisionGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Foo
version = v1.0.0
basepath = /rpc

service Bar
  - Ping()

service FooBar
  - Pong()
`

	output := generateSwift(t, schema)
	requireContains(t, output, "public enum FooBarAPI")
	requireContains(t, output, "public struct FooBarClient: Sendable")
	requireContains(t, output, "public enum FooBarServiceAPI")
	requireContains(t, output, "public struct FooBarServiceClient: Sendable")
}

func TestWebrpcHeaderOptionGeneration(t *testing.T) {
	schema := `
webrpc = v1

name = Headers
version = v1.0.0
basepath = /rpc

service Headers
  - Ping() => (ok: bool)
`

	defaultOutput := generateSwift(t, schema)
	requireContains(t, defaultOutput, `public let WEBRPC_HEADER = "Webrpc"`)
	requireContains(t, defaultOutput, "includeWebRPCHeader: true")
	requireContains(t, defaultOutput, "requestHeaders[WEBRPC_HEADER] = WEBRPC_HEADER_VALUE")

	disabledOutput := generateSwift(t, schema, "-webrpcHeader=false")
	requireContains(t, disabledOutput, `public let WEBRPC_HEADER = "Webrpc"`)
	requireContains(t, disabledOutput, "includeWebRPCHeader: false")
	requireNotContains(t, disabledOutput, "requestHeaders[WEBRPC_HEADER] = WEBRPC_HEADER_VALUE")
}

func TestExternalSchemaGeneratesCompilableClient(t *testing.T) {
	if os.Getenv("WEBRPC_SWIFT_EXTERNAL_SCHEMA") == "" {
		t.Skip("set WEBRPC_SWIFT_EXTERNAL_SCHEMA=1 to run external schema integration test")
	}

	schemaPath := filepath.Join("..", "waas", "proto", "waas.ridl")
	if _, err := os.Stat(schemaPath); err != nil {
		t.Skipf("external schema fixture not available: %v", err)
	}

	output := generateSwiftFromSchemaFile(t, schemaPath)

	requireContains(t, output, "public struct CommitVerifierRequest: Codable, Sendable")
	requireContains(t, output, "public struct RevokeAccessRequest: Codable, Sendable")
	requireContains(t, output, `public let WEBRPC_HEADER = "Webrpc"`)
	requireContains(t, output, "public struct WebRPCGeneratedVersions: Sendable, Equatable")
	requireContains(t, output, `public static let path = "/CommitVerifier"`)
	requireContains(t, output, "public func commitVerifier(")
	requireContains(t, output, "public func revokeAccess(")
	requireContains(t, output, "public let metadata: [String: String]")
	requireContains(t, output, "public let authKey: Data?")
	requireContains(t, output, "public let deviceKey: Data?")
	requireContains(t, output, "public let privateKey: Data")
	requireContains(t, output, "case idToken")
	requireContains(t, output, "case authCodePkce")
	requireContains(t, output, "case ethereumEoa")
	requireContains(t, output, "case oauthError")

	project := writeSwiftPackage(t, "external-schema", map[string]string{
		"Sources/Generated/Client.swift": output,
		"Tests/GeneratedTests/GeneratedTests.swift": `
import Foundation
import XCTest
@testable import Generated

final class GeneratedTests: XCTestCase {
    func testCommitVerifierEncodesSemanticPayload() throws {
        let body = try WaasWalletAPI.CommitVerifier.encodeRequest(
            CommitVerifierRequest(
                identityType: .email,
                authMode: .idToken,
                metadata: ["region": "eu", "tier": "gold"],
                handle: "user@example.com"
            )
        )

        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertEqual(json["identityType"] as? String, "Email")
        XCTAssertEqual(json["authMode"] as? String, "IDToken")
        XCTAssertEqual(json["handle"] as? String, "user@example.com")

        let metadata = try XCTUnwrap(json["metadata"] as? [String: String])
        XCTAssertEqual(metadata["region"], "eu")
        XCTAssertEqual(metadata["tier"], "gold")
    }

    func testSendTransactionEncodesSemanticPayload() throws {
        let body = try WaasWalletAPI.SendTransaction.encodeRequest(
            SendTransactionRequest(
                network: "amoy",
                wallet: "0xwallet",
                to: "0xabc",
                value: "0",
                data: "0x1234",
                mode: .native,
                feeCeiling: "1000000",
                nonce: "42"
            )
        )

        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertEqual(json["network"] as? String, "amoy")
        XCTAssertEqual(json["wallet"] as? String, "0xwallet")
        XCTAssertEqual(json["to"] as? String, "0xabc")
        XCTAssertEqual(json["value"] as? String, "0")
        XCTAssertEqual(json["data"] as? String, "0x1234")
        XCTAssertEqual(json["mode"] as? String, "Native")
        XCTAssertEqual(json["feeCeiling"] as? String, "1000000")
        XCTAssertEqual(json["nonce"] as? String, "42")
    }

    func testVersionHeaderParsingWorksForExternalSchema() {
        let versions = versionFromHeader(["webrpc": WEBRPC_HEADER_VALUE])
        XCTAssertEqual(versions.schemaName, "waas")
        XCTAssertEqual(versions.schemaVersion, "v0.1.0")
        XCTAssertFalse(versions.webrpcGenVersion.isEmpty)
        XCTAssertTrue(versions.codeGenName.hasPrefix("gen-swift"))
        XCTAssertFalse(versions.codeGenVersion.isEmpty)
    }
}
`,
	})

	runSwiftTest(t, project)
}

func generateSwift(t *testing.T, schema string, extraArgs ...string) string {
	t.Helper()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.ridl")
	outPath := filepath.Join(dir, "Client.swift")

	if err := os.WriteFile(schemaPath, []byte(strings.TrimSpace(schema)+"\n"), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	args := []string{
		"-schema=" + schemaPath,
		"-target=" + repoRoot(t),
		"-client",
	}
	args = append(args, extraArgs...)
	args = append(args, "-out="+outPath)

	runWebrpcGen(t, args...)

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated output: %v", err)
	}
	return string(content)
}

func generateSwiftErr(t *testing.T, schema string, extraArgs ...string) string {
	t.Helper()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.ridl")
	outPath := filepath.Join(dir, "Client.swift")

	if err := os.WriteFile(schemaPath, []byte(strings.TrimSpace(schema)+"\n"), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	args := []string{
		"-schema=" + schemaPath,
		"-target=" + repoRoot(t),
	}
	args = append(args, extraArgs...)
	args = append(args, "-out="+outPath)

	return runWebrpcGenErr(t, args...)
}

func generateSwiftFromSchemaFile(t *testing.T, schemaPath string, extraArgs ...string) string {
	t.Helper()

	outPath := filepath.Join(t.TempDir(), "Client.swift")
	absSchemaPath, err := filepath.Abs(schemaPath)
	if err != nil {
		t.Fatalf("abs schema path: %v", err)
	}

	args := []string{
		"-schema=" + absSchemaPath,
		"-target=" + repoRoot(t),
		"-client",
	}
	args = append(args, extraArgs...)
	args = append(args, "-out="+outPath)

	runWebrpcGen(t, args...)

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated output: %v", err)
	}
	return string(content)
}

func writeSwiftPackage(t *testing.T, name string, files map[string]string) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	hasTests := false
	for relPath := range files {
		if strings.HasPrefix(relPath, "Tests/") {
			hasTests = true
			break
		}
	}

	targetsBlock := `    targets: [
        .target(name: "Generated"),`
	if hasTests {
		targetsBlock += `
        .testTarget(name: "GeneratedTests", dependencies: ["Generated"]),`
	}
	targetsBlock += `
    ]`

	packageSwift := fmt.Sprintf(`// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: %q,
    platforms: [
        .macOS(.v12),
        .iOS(.v15),
    ],
    products: [
        .library(name: "Generated", targets: ["Generated"]),
    ],
%s
)
`, name, targetsBlock)

	if err := os.WriteFile(filepath.Join(dir, "Package.swift"), []byte(packageSwift), 0o644); err != nil {
		t.Fatalf("write Package.swift: %v", err)
	}

	for relPath, content := range files {
		target := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", target, err)
		}
		if err := os.WriteFile(target, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", target, err)
		}
	}

	return dir
}

func runSwiftTest(t *testing.T, projectDir string) {
	t.Helper()

	cmd := exec.Command("swift", "test")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("swift test failed: %v\n%s", err, output)
	}
}

func runSwiftBuild(t *testing.T, projectDir string) {
	t.Helper()

	cmd := exec.Command("swift", "build")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("swift build failed: %v\n%s", err, output)
	}
}

func runWebrpcGen(t *testing.T, args ...string) {
	t.Helper()

	cmdName, prefix, workDir := webrpcGenCommand(t)
	cmd := exec.Command(cmdName, append(prefix, args...)...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("webrpc-gen failed: %v\n%s", err, output)
	}
}

func runWebrpcGenErr(t *testing.T, args ...string) string {
	t.Helper()

	cmdName, prefix, workDir := webrpcGenCommand(t)
	cmd := exec.Command(cmdName, append(prefix, args...)...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected webrpc-gen to fail")
	}
	return string(output)
}

func webrpcGenCommand(t *testing.T) (string, []string, string) {
	t.Helper()

	if bin := os.Getenv("WEBRPC_GEN_BIN"); bin != "" {
		return bin, nil, repoRoot(t)
	}
	return "go", []string{"-C", filepath.Join(repoRoot(t), "tools"), "tool", "webrpc-gen"}, repoRoot(t)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return dir
}

func requireContains(t *testing.T, text, needle string) {
	t.Helper()
	if !strings.Contains(text, needle) {
		t.Fatalf("expected output to contain %q\n\n%s", needle, text)
	}
}

func requireNotContains(t *testing.T, text, needle string) {
	t.Helper()
	if strings.Contains(text, needle) {
		t.Fatalf("expected output not to contain %q\n\n%s", needle, text)
	}
}

func requireRegexp(t *testing.T, expr, text string) {
	t.Helper()
	if !regexp.MustCompile(expr).MatchString(text) {
		t.Fatalf("expected output to match %q\n\n%s", expr, text)
	}
}
