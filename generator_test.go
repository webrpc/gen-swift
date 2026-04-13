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

	project := writeSwiftPackage(t, "keyword-fields", map[string]string{
		"Sources/Generated/Generated.swift": output,
	})

	runSwiftBuild(t, project)
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

	project := writeSwiftPackage(t, "null-optional", map[string]string{
		"Sources/Generated/Generated.swift": output,
	})

	runSwiftBuild(t, project)
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

	project := writeSwiftPackage(t, "service-collision", map[string]string{
		"Sources/Generated/Generated.swift": output,
	})

	runSwiftBuild(t, project)
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

	project := writeSwiftPackage(t, "multi-service-formatting", map[string]string{
		"Sources/Generated/Generated.swift": output,
	})

	runSwiftBuild(t, project)
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

	project := writeSwiftPackage(t, "multi-service-collision", map[string]string{
		"Sources/Generated/Generated.swift": output,
	})

	runSwiftBuild(t, project)
}

func TestSwiftPackageCompilesAndRunsRuntimeHelpers(t *testing.T) {
	schema := `
webrpc = v1

name = Helper
version = v1.0.0
basepath = /rpc

service Helper
  - GetUser(userId: uint64) => (code: uint32, username: string)

error 200 UserNotFound "user not found"
`

	output := generateSwift(t, schema)

	project := writeSwiftPackage(t, "helper-runtime", map[string]string{
		"Sources/Generated/Generated.swift": output,
		"Tests/GeneratedTests/GeneratedTests.swift": `
import Foundation
import XCTest
@testable import Generated

final class GeneratedTests: XCTestCase {
    func testHelperRoundTripWorks() throws {
        let request = HelperAPI.GetUser.Request(userId: 7)
        let body = try HelperAPI.GetUser.encodeRequest(request)
        let bodyJSON = try JSONSerialization.jsonObject(with: body) as? [String: Any]
        XCTAssertEqual(bodyJSON?["userId"] as? NSNumber, 7)

        let response = try HelperAPI.GetUser.decodeResponse(Data(#"{"code":200,"username":"alice"}"#.utf8))
        XCTAssertEqual(response.code, 200)
        XCTAssertEqual(response.username, "alice")

        XCTAssertEqual(HelperAPI.GetUser.path, "/GetUser")
        XCTAssertEqual(HelperAPI.GetUser.urlPath, "/rpc/Helper/GetUser")
    }

    func testErrorDecodeWorks() throws {
        let error = decodeWebRPCError(
            statusCode: 404,
            data: Data(#"{"error":"UserNotFound","code":200,"msg":"user not found","cause":"","status":404}"#.utf8)
        )

        XCTAssertEqual(error.error, "UserNotFound")
        XCTAssertEqual(error.code, 200)
        XCTAssertEqual(error.message, "user not found")
        XCTAssertEqual(error.status, 404)
        switch error.kind {
        case .userNotFound:
            break
        default:
            XCTFail("expected schema error kind")
        }
    }

    func testClientSendsWebrpcHeaderAndParsesVersions() async throws {
        actor Recorder {
            private(set) var headers: [String: String] = [:]

            func set(headers: [String: String]) {
                self.headers = headers
            }

            func snapshot() -> [String: String] {
                headers
            }
        }

        struct CaptureTransport: WebRPCTransport {
            let recorder: Recorder

            func post(
                baseURL: String,
                path: String,
                body: Data,
                headers: [String: String]
            ) async throws -> WebRPCHTTPResponse {
                await recorder.set(headers: headers)
                return WebRPCHTTPResponse(
                    statusCode: 200,
                    body: Data(#"{"code":200,"username":"alice"}"#.utf8),
                    headers: ["webrpc": WEBRPC_HEADER_VALUE]
                )
            }
        }

        let recorder = Recorder()
        let client = HelperClient(
            baseURL: "https://example.com",
            transport: CaptureTransport(recorder: recorder),
            headers: { ["X-Test": "1"] }
        )

        let response = try await client.getUser(HelperAPI.GetUser.Request(userId: 9))
        XCTAssertEqual(response.username, "alice")

        let sentHeaders = await recorder.snapshot()
        XCTAssertEqual(sentHeaders[WEBRPC_HEADER], WEBRPC_HEADER_VALUE)
        XCTAssertEqual(sentHeaders["X-Test"], "1")

        let versions = versionFromHeader(["webrpc": WEBRPC_HEADER_VALUE])
        XCTAssertFalse(versions.webrpcGenVersion.isEmpty)
        XCTAssertEqual(versions.schemaName, "Helper")
        XCTAssertEqual(versions.schemaVersion, "v1.0.0")

        let responseVersions = versionFromHeader(
            WebRPCHTTPResponse(statusCode: 200, body: Data(), headers: ["Webrpc": WEBRPC_HEADER_VALUE])
        )
        XCTAssertEqual(responseVersions, versions)
    }
}
`,
	})

	runSwiftTest(t, project)
}

func TestAnyIntegerRoundTripPreservesPrecision(t *testing.T) {
	schema := `
webrpc = v1

name = AnyValue
version = v1.0.0
basepath = /rpc

struct Box
  - payload: any

service AnyValue
  - Echo(Box) => (Box)
`

	output := generateSwift(t, schema)

	project := writeSwiftPackage(t, "any-integer", map[string]string{
		"Sources/Generated/Generated.swift": output,
		"Tests/GeneratedTests/GeneratedTests.swift": `
import Foundation
import XCTest
@testable import Generated

final class GeneratedTests: XCTestCase {
    func testLargeIntegerRoundTripPreservesPrecision() throws {
        let source = Data(#"{"payload":9007199254740993}"#.utf8)
        let decoded = try JSONDecoder().decode(Box.self, from: source)

        switch decoded.payload {
        case .integer(let value):
            XCTAssertEqual(value, 9_007_199_254_740_993)
        default:
            XCTFail("expected integer payload")
        }

        let encoded = try JSONEncoder().encode(decoded)
        let encodedString = String(decoding: encoded, as: UTF8.self)
        XCTAssertTrue(encodedString.contains(#""payload":9007199254740993"#))
    }
}
`,
	})

	runSwiftTest(t, project)
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

	disabledOutput := generateSwift(t, schema, "-webrpcHeader=false")
	requireContains(t, disabledOutput, `public let WEBRPC_HEADER = "Webrpc"`)
	requireContains(t, disabledOutput, "includeWebRPCHeader: false")

	project := writeSwiftPackage(t, "webrpc-header-disabled", map[string]string{
		"Sources/Generated/Generated.swift": disabledOutput,
		"Tests/GeneratedTests/GeneratedTests.swift": `
import Foundation
import XCTest
@testable import Generated

final class GeneratedTests: XCTestCase {
    func testDisabledHeaderIsNotAdded() async throws {
        actor Recorder {
            private(set) var headers: [String: String] = [:]

            func set(headers: [String: String]) {
                self.headers = headers
            }

            func snapshot() -> [String: String] {
                headers
            }
        }

        struct CaptureTransport: WebRPCTransport {
            let recorder: Recorder

            func post(
                baseURL: String,
                path: String,
                body: Data,
                headers: [String: String]
            ) async throws -> WebRPCHTTPResponse {
                await recorder.set(headers: headers)
                return WebRPCHTTPResponse(
                    statusCode: 200,
                    body: Data(#"{"ok":true}"#.utf8)
                )
            }
        }

        let recorder = Recorder()
        let client = HeadersClient(
            baseURL: "https://example.com",
            transport: CaptureTransport(recorder: recorder)
        )

        let response = try await client.ping()
        XCTAssertEqual(response.ok, true)

        let sentHeaders = await recorder.snapshot()
        XCTAssertNil(sentHeaders[WEBRPC_HEADER])
    }
}
`,
	})

	runSwiftTest(t, project)
}

func TestExternalSchemaGeneratesCompilableClient(t *testing.T) {
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
	})

	runSwiftBuild(t, project)
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
