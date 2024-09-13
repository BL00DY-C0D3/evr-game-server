module github.com/BL00DY-C0D3/evr-game-wrapper/standalone-agent

go 1.15

replace github.com/BL00DY-C0D3/evr-game-wrapper => ../

require go.uber.org/zap v1.27.0

require github.com/BL00DY-C0D3/evr-game-wrapper v0.0.0-00010101000000-000000000000

replace go.uber.org/zap v1.27.0 => go.uber.org/zap v1.16.0

replace go.uber.org/multierr v1.10.0 => go.uber.org/multierr v1.9.0
