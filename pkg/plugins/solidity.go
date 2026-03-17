package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SolidityPlugin provides Solidity development capabilities
type SolidityPlugin struct {
	workDir      string
	solcPath     string
	solcVersion  string
}

// NewSolidityPlugin creates a new Solidity plugin
func NewSolidityPlugin(workDir string) (*SolidityPlugin, error) {
	// Find solc compiler
	solcPath, err := exec.LookPath("solc")
	if err != nil {
		return nil, fmt.Errorf("solc compiler not found in PATH: %w\nInstall with: npm install -g solc", err)
	}

	// Get version
	cmd := exec.Command(solcPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get solc version: %w", err)
	}

	version := strings.TrimSpace(string(output))

	return &SolidityPlugin{
		workDir:     workDir,
		solcPath:    solcPath,
		solcVersion: version,
	}, nil
}

// CompileContract compiles a Solidity contract
func (sp *SolidityPlugin) CompileContract(contractPath string) (*CompilationResult, error) {
	// Verify file exists
	if _, err := os.Stat(contractPath); err != nil {
		return nil, fmt.Errorf("contract file not found: %w", err)
	}

	// Run solc compiler
	cmd := exec.Command(
		sp.solcPath,
		"--combined-json", "abi,bin,metadata",
		"--optimize",
		"--optimize-runs", "200",
		contractPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	// Parse output
	result := &CompilationResult{
		Success:    true,
		Output:     string(output),
		SourceFile: contractPath,
	}

	return result, nil
}

// CompilationResult represents compilation output
type CompilationResult struct {
	Success    bool
	Output     string
	ABI        string
	Bytecode   string
	Metadata   string
	SourceFile string
	Errors     []string
	Warnings   []string
}

// ValidateSyntax checks Solidity syntax without compiling
func (sp *SolidityPlugin) ValidateSyntax(contractPath string) ([]string, error) {
	cmd := exec.Command(sp.solcPath, "--ast-compact-json", contractPath)
	output, err := cmd.CombinedOutput()

	errors := []string{}
	if err != nil {
		// Parse error output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Error:") || strings.Contains(line, "Warning:") {
				errors = append(errors, line)
			}
		}
	}

	return errors, nil
}

// FormatCode formats Solidity code (using prettier-plugin-solidity if available)
func (sp *SolidityPlugin) FormatCode(contractPath string) error {
	// Check if prettier with solidity plugin is available
	prettierPath, err := exec.LookPath("prettier")
	if err != nil {
		return fmt.Errorf("prettier not found (optional): %w", err)
	}

	cmd := exec.Command(prettierPath, "--write", "--plugin=prettier-plugin-solidity", contractPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	return nil
}

// GetContractName extracts the contract name from source file
func (sp *SolidityPlugin) GetContractName(contractPath string) (string, error) {
	content, err := os.ReadFile(contractPath)
	if err != nil {
		return "", err
	}

	// Simple regex-like parsing to find "contract ContractName"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "contract ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return strings.TrimSuffix(parts[1], "{"), nil
			}
		}
	}

	return "", fmt.Errorf("no contract declaration found")
}

// CreateTemplate creates a new Solidity contract from template
func (sp *SolidityPlugin) CreateTemplate(contractName, templateType string) (string, error) {
	var template string

	switch templateType {
	case "erc20":
		template = generateERC20Template(contractName)
	case "erc721":
		template = generateERC721Template(contractName)
	case "basic":
		template = generateBasicTemplate(contractName)
	default:
		return "", fmt.Errorf("unknown template type: %s", templateType)
	}

	// Create file
	filename := filepath.Join(sp.workDir, contractName+".sol")
	if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return filename, nil
}

func generateBasicTemplate(name string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract %s {
    address public owner;

    constructor() {
        owner = msg.sender;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    // Add your functions here
}
`, name)
}

func generateERC20Template(name string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract %s is ERC20, Ownable {
    constructor() ERC20("%s", "TKN") Ownable(msg.sender) {
        _mint(msg.sender, 1000000 * 10 ** decimals());
    }

    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
}
`, name, name)
}

func generateERC721Template(name string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract %s is ERC721, Ownable {
    uint256 private _tokenIdCounter;

    constructor() ERC721("%s", "NFT") Ownable(msg.sender) {}

    function safeMint(address to) public onlyOwner {
        uint256 tokenId = _tokenIdCounter++;
        _safeMint(to, tokenId);
    }
}
`, name, name)
}

// SyntaxHighlighter provides basic Solidity syntax highlighting rules
type SyntaxHighlighter struct {
	Keywords     []string
	Types        []string
	Builtins     []string
}

// NewSolidityHighlighter returns Solidity syntax rules
func NewSolidityHighlighter() *SyntaxHighlighter {
	return &SyntaxHighlighter{
		Keywords: []string{
			"contract", "interface", "library", "abstract", "is",
			"function", "modifier", "event", "struct", "enum",
			"public", "private", "internal", "external",
			"pure", "view", "payable", "constant",
			"if", "else", "for", "while", "do", "break", "continue", "return",
			"require", "assert", "revert",
			"new", "delete", "emit",
			"mapping", "memory", "storage", "calldata",
			"constructor", "fallback", "receive",
			"pragma", "import", "using",
		},
		Types: []string{
			"uint", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256",
			"int", "int8", "int16", "int32", "int64", "int128", "int256",
			"bool", "address", "bytes", "bytes1", "bytes32",
			"string",
		},
		Builtins: []string{
			"msg", "block", "tx", "now",
			"this", "super",
			"abi", "keccak256", "sha256",
			"ecrecover", "addmod", "mulmod",
		},
	}
}
