# 阿里云百炼 models

> 来源: https://help.aliyun.com/zh/model-studio/models

阿里云百炼提供千问及第三方模型服务，覆盖文本、图像、音频、视频等多种模态。

    
      
      
        
          .bl-models {
            --blue: #0070CC;
            --blue-2: #0058A3;
            --blue-3: #E6F1FA;
            --ink: #1F2329;
            --muted: #6C7377;
            --muted-2: #909499;
            --rule: #E4E7ED;
            --bg: #FFFFFF;
            --bg-soft: #F7F8FA;
            --radius: 8px;
            font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial, sans-serif;
            color: var(--ink);
            font-size: 14px;
            line-height: 1.6
          }
          .bl-models * {
            box-sizing: border-box
          }
          .bl-models a {
            color: inherit;
            text-decoration: none
          }
          .bl-models .bl-sec {
            margin: 0 !important;
            padding-top: 32px !important;
            border-top: 1px solid #E4E7ED !important
          }
          .bl-models .bl-sec:first-of-type {
            padding-top: 32px !important;
            border-top: 1px solid #E4E7ED !important;
            margin-top: 24px !important
          }
          .bl-models .bl-sec+.bl-sec {
            margin-top: 32px !important
          }
          .bl-models .bl-sec-head {
            display: flex;
            align-items: baseline;
            gap: 14px;
            margin: 0 0 20px;
            padding: 0;
            position: relative;
            border: none
          }
          .bl-models .bl-sec-head::after {
            display: none
          }
          .bl-models .bl-sec-head h2 {
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            color: var(--ink);
            letter-spacing: -.005em
          }
          .bl-models .bl-sub {
            margin: 24px 0 0 !important;
            padding-top: 0 !important;
            border-top: none !important
          }
          .bl-models .bl-sub:first-of-type {
            margin-top: 8px !important
          }
          .bl-models .bl-sub-head {
            position: relative;
            display: flex;
            flex-direction: column;
            align-items: flex-start;
            gap: 6px;
            margin: 0 0 16px
          }
          .bl-models .bl-sub-head h3 {
            margin: 0;
            font-size: 15px;
            font-weight: 600;
            color: var(--ink)
          }
          .bl-models .bl-sub-head .bl-sub-desc {
            font-size: 13px;
            color: var(--muted);
            margin: 0;
            line-height: 1.5
          }
          .bl-models .bl-more {
            position: absolute;
            top: 0;
            right: 0;
            font-size: 13px;
            color: var(--blue);
            white-space: nowrap;
            display: inline-flex;
            align-items: center;
            gap: 4px
          }
          .bl-models .bl-more:hover {
            color: var(--blue-2);
            text-decoration: underline
          }
          .bl-models .bl-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
            gap: 12px
          }
          .bl-models .bl-card {
            position: relative;
            display: flex;
            flex-direction: column;
            gap: 8px;
            padding: 14px 16px;
            background: var(--bg);
            border: 1px solid var(--rule);
            border-radius: var(--radius);
            transition: border-color .18s ease, box-shadow .18s ease, transform .18s ease;
            min-width: 0;
            overflow: hidden;
            box-sizing: border-box
          }
          .bl-models .bl-card:hover {
            border-color: var(--blue);
            box-shadow: 0 4px 16px -4px rgba(0, 112, 204, .18), 0 1px 4px rgba(0, 112, 204, .06);
            transform: translateY(-1px)
          }
          .bl-models .bl-card-head {
            display: flex;
            align-items: center;
            gap: 10px
          }
          .bl-models .bl-icon {
            width: 45px;
            height: 45px;
            flex-shrink: 0;
            display: inline-flex;
            align-items: center;
            justify-content: center
          }
          .bl-models .bl-icon img {
            width: 100%;
            height: 100%;
            object-fit: contain;
            display: block
          }
          .bl-models .bl-card-name {
            flex: 1;
            min-width: 0;
            font-size: 14px;
            font-weight: 600;
            color: var(--ink);
            letter-spacing: -.005em;
            font-family: "SF Mono", "Menlo", "Consolas", ui-monospace, monospace;
            line-height: 1.35;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
            word-break: break-all
          }
          .bl-models .bl-card:hover .bl-card-name {
            color: var(--blue)
          }
          .bl-models .bl-vendor {
            position: absolute;
            top: 0;
            right: 0;
            color: #fff;
            font-size: 10px;
            font-weight: 500;
            padding: 3px 8px 3px 12px;
            line-height: 1.4;
            letter-spacing: .04em;
            clip-path: polygon(8px 0, 100% 0, 100% 100%, 0 100%);
            border-top-right-radius: var(--radius);
            z-index: 1;
            pointer-events: none
          }

          .bl-models .bl-vendor.qwen {
            background: linear-gradient(90deg, #FF9A5B, #FF5000)
          }

          .bl-models .bl-vendor.third {
            background: linear-gradient(90deg, #66A9FF, #1E6FFF)
          }
          .bl-models .bl-card-desc {
            margin: 0;
            font-size: 13px;
            color: var(--muted);
            line-height: 1.55;
            flex: 1
          }
          .bl-models .bl-card-foot {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 8px;
            padding-top: 4px
          }
          .bl-models .bl-badges {
            display: flex;
            gap: 4px;
            flex-wrap: wrap
          }
          .bl-models .bl-badge {
            font-size: 10px;
            color: var(--muted);
            background: var(--bg-soft);
            padding: 2px 6px;
            border-radius: 3px;
            letter-spacing: .02em;
            line-height: 1.4
          }
          .bl-models .bl-badge.hot {
            color: var(--blue);
            background: var(--blue-3)
          }
          .bl-models .bl-arrow {
            display: inline-flex;
            align-items: center;
            color: var(--muted-2);
            transition: color .18s, transform .18s;
            font-size: 14px
          }
          .bl-models .bl-card:hover .bl-arrow {
            color: var(--blue);
            transform: translateX(2px)
          }
          .bl-models .bl-meta {
            font-size: 10px;
            color: var(--muted);
            font-family: "SF Mono", "Menlo", ui-monospace, monospace;
            letter-spacing: .01em;
            min-width: 0;
            flex: 1;
            line-height: 1.5;
            white-space: normal;
            word-break: break-word
          }
          .bl-models .bl-mods {
            display: inline-flex;
            gap: 4px;
            flex-shrink: 0;
            align-items: center;
            margin-left: auto
          }
          .bl-models .bl-mod {
            width: 20px;
            height: 20px;
            border-radius: 4px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0
          }
          .bl-models .bl-mod svg {
            width: 11px;
            height: 11px;
            display: block
          }
          .bl-models .bl-mod-text {
            color: #1677FF;
            background: #E6F1FA
          }
          .bl-models .bl-mod-image {
            color: #37A55C;
            background: #E6F4EC
          }
          .bl-models .bl-mod-embed {
            color: #7B5BD6;
            background: #EFE9FA
          }
          .bl-models .bl-mod-audio {
            color: #8B5CF6;
            background: #F1ECFB
          }
          .bl-models .bl-mod-video {
            color: #E8841A;
            background: #FCF1E1
          }
          .bl-models .bl-mod-s2s {
            color: #E8841A;
            background: #FCF1E1
          }
          .bl-models .bl-mod-tts {
            color: #8B5CF6;
            background: #F1ECFB
          }
          .bl-models .bl-mod-asr {
            color: #2EA0BD;
            background: #E0F2F4
          }
          .bl-models .bl-mod-vision {
            color: #0070CC;
            background: #E6F1FA
          }
          .bl-models .bl-mod-rerank {
            color: #2EA0BD;
            background: #E0F2F4
          }
          .bl-models .bl-mod-speech {
            color: #6E5BC0;
            background: #ECE7F7
          }
          .bl-models .bl-mod-3d {
            color: #D4621A;
            background: #FAEBE0
          }
          .bl-models .bl-footer {
            margin: 48px 0 8px;
            padding: 24px 28px;
            background: var(--bg-soft);
            border: 1px solid var(--rule);
            border-radius: var(--radius);
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 16px;
            flex-wrap: wrap
          }
          .bl-models .bl-footer-text h3 {
            margin: 0 0 4px;
            font-size: 16px;
            font-weight: 600;
            color: var(--ink)
          }
          .bl-models .bl-footer-text p {
            margin: 0;
            font-size: 13px;
            color: var(--muted)
          }
          .bl-models .bl-btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 9px 18px;
            background: var(--blue);
            color: #fff;
            border-radius: var(--radius);
            font-size: 13px;
            font-weight: 500;
            transition: background .18s ease
          }
          .bl-models .bl-btn:hover {
            background: var(--blue-2)
          }
          @media (max-width:960px) {
            .bl-models .bl-grid {
              display: grid;
              grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
              gap: 12px
            }
            .bl-models .bl-sub-head .bl-sub-desc {
              margin-left: 0;
              flex-basis: 100%
            }
          }
          .bl-models h1,
          .bl-models h2,
          .bl-models h3,
          .bl-models h4 {
            border: none !important;
            background: none !important;
            padding: 0 !important;
            margin: 0 !important;
            line-height: 1.3 !important
          }
          .bl-models p {
            margin: 0 !important
          }
          .bl-cardwrap { position: relative; display: block; width: 100%; min-width: 0; box-sizing: border-box; }
          .bl-models .bl-grid > .bl-cardwrap { width: 100% !important; min-width: 0 !important; max-width: 100% !important; }
          .bl-models .bl-grid > .bl-cardwrap > .bl-card { width: 100% !important; height: auto !important; box-sizing: border-box; min-width: 0; }
          .bl-models .bl-grid > .bl-cardwrap { align-self: start; }  
          .bl-cardwrap:hover { z-index: 5; }
          .bl-cardwrap .bl-pop { position: absolute !important; }
          .bl-pop {
            position: absolute; left: 0; right: 0; top: calc(100% + 10px); z-index: 30;
            width: 100%; box-sizing: border-box;
            background: #fff; border: 1px solid var(--rule); border-radius: 12px;
            box-shadow: 0 14px 40px rgba(31,35,41,.18), 0 2px 8px rgba(31,35,41,.06);
            text-align: left; white-space: normal;
            font-size: 12px; line-height: 1.55; color: var(--ink);
            opacity: 0; visibility: hidden; transform: translateY(-6px);
            transition: opacity .18s ease, transform .18s ease, visibility .18s;
          }
          .bl-cardwrap:hover .bl-pop { opacity: 1; visibility: visible; transform: none; }
          .bl-rcol > .bl-pop { left: auto !important; right: 0 !important; }
          .markdown-body { overflow: visible !important; }
          .bl-pop .bl-pop-t {
            font-weight: 600; font-size: 13px; color: var(--ink);
            padding: 11px 16px 9px; border-bottom: 1px solid var(--rule);
            background: linear-gradient(180deg, var(--blue-3), #fff);
            display: flex; align-items: center; gap: 8px; border-radius: 12px 12px 0 0;
          }
          .bl-pop .bl-pop-t::before {
            content: ""; width: 7px; height: 7px; border-radius: 50%;
            background: var(--blue); box-shadow: 0 0 0 3px rgba(0,112,204,.15); flex: none;
          }
          .bl-pop input.bl-rtab { position: absolute; opacity: 0; pointer-events: none; }
          .bl-pop .bl-tabs { display: flex; gap: 2px; padding: 8px 12px 0; border-bottom: 1px solid var(--rule); flex-wrap: wrap; }
          .bl-pop .bl-tabs label {
            font-size: 11px; padding: 5px 9px; border-radius: 6px 6px 0 0; cursor: pointer;
            color: var(--muted); border: 1px solid transparent; border-bottom: none; margin-bottom: -1px;
          }
          .bl-pop .bl-tabs label:hover { color: var(--ink); }
          .bl-pop .bl-tabc { display: none; padding: 12px 16px 14px; }
          .bl-pop input.bl-rtab:nth-of-type(1):checked ~ .bl-tabs label:nth-of-type(1),
          .bl-pop input.bl-rtab:nth-of-type(2):checked ~ .bl-tabs label:nth-of-type(2),
          .bl-pop input.bl-rtab:nth-of-type(3):checked ~ .bl-tabs label:nth-of-type(3),
          .bl-pop input.bl-rtab:nth-of-type(4):checked ~ .bl-tabs label:nth-of-type(4),
          .bl-pop input.bl-rtab:nth-of-type(5):checked ~ .bl-tabs label:nth-of-type(5),
          .bl-pop input.bl-rtab:nth-of-type(6):checked ~ .bl-tabs label:nth-of-type(6),
          .bl-pop input.bl-rtab:nth-of-type(7):checked ~ .bl-tabs label:nth-of-type(7),
          .bl-pop input.bl-rtab:nth-of-type(8):checked ~ .bl-tabs label:nth-of-type(8) {
            color: var(--blue); font-weight: 600; background: #fff;
            border-color: var(--rule); border-bottom: 1px solid #fff;
          }
          .bl-pop input.bl-rtab:nth-of-type(1):checked ~ .bl-tabc:nth-of-type(1),
          .bl-pop input.bl-rtab:nth-of-type(2):checked ~ .bl-tabc:nth-of-type(2),
          .bl-pop input.bl-rtab:nth-of-type(3):checked ~ .bl-tabc:nth-of-type(3),
          .bl-pop input.bl-rtab:nth-of-type(4):checked ~ .bl-tabc:nth-of-type(4),
          .bl-pop input.bl-rtab:nth-of-type(5):checked ~ .bl-tabc:nth-of-type(5),
          .bl-pop input.bl-rtab:nth-of-type(6):checked ~ .bl-tabc:nth-of-type(6),
          .bl-pop input.bl-rtab:nth-of-type(7):checked ~ .bl-tabc:nth-of-type(7),
          .bl-pop input.bl-rtab:nth-of-type(8):checked ~ .bl-tabc:nth-of-type(8) { display: block; }
          .bl-tabc input.bl-ptab { position: absolute; opacity: 0; pointer-events: none; }
          .bl-tabc .bl-ptabs { display: flex; gap: 4px; margin: 12px 0 9px; flex-wrap: wrap; }
          .bl-tabc .bl-ptabs label {
            font-size: 10px; padding: 3px 7px; border-radius: 6px; cursor: pointer; white-space: nowrap;
            color: var(--muted); background: var(--blue-3); border: 1px solid transparent;
          }
          .bl-tabc .bl-ptabs label:hover { color: var(--ink); }
          .bl-tabc .bl-ptabc { display: none; }
          .bl-tabc input.bl-ptab:nth-of-type(1):checked ~ .bl-ptabs label:nth-of-type(1),
          .bl-tabc input.bl-ptab:nth-of-type(2):checked ~ .bl-ptabs label:nth-of-type(2),
          .bl-tabc input.bl-ptab:nth-of-type(3):checked ~ .bl-ptabs label:nth-of-type(3) {
            color: var(--blue); font-weight: 600; background: #fff; border-color: var(--blue);
          }
          .bl-tabc input.bl-ptab:nth-of-type(1):checked ~ .bl-ptabc:nth-of-type(1),
          .bl-tabc input.bl-ptab:nth-of-type(2):checked ~ .bl-ptabc:nth-of-type(2),
          .bl-tabc input.bl-ptab:nth-of-type(3):checked ~ .bl-ptabc:nth-of-type(3) { display: block; }
          .bl-pop .bl-pop-row { margin-top: 10px; }
          .bl-pop .bl-pop-row:first-child { margin-top: 0; }
          .bl-pop .bl-pop-k { display: block; color: var(--muted); font-size: 11px; margin-bottom: 3px; }
          .bl-pop code {
            display: inline-block; background: var(--bg-soft); border: 1px solid var(--rule);
            padding: 3px 9px; border-radius: 6px; color: var(--blue-2);
            font-family: ui-monospace, Menlo, monospace; font-size: 11.5px; word-break: break-all; line-height: 1.5;
          }
          .bl-pop a.bl-pop-link { color: var(--blue); text-decoration: none; }
          .bl-pop a.bl-pop-link:hover { text-decoration: underline; }
          .bl-pop pre.bl-json {
            margin: 9px 0 0; padding: 9px 11px; background: #0b0d11; color: #b6c2cf;
            border-radius: 7px; font-family: ui-monospace, Menlo, monospace; font-size: 10.5px;
            line-height: 1.5; white-space: pre-wrap; word-break: break-all; user-select: all;
          }
          .bl-pop .bl-json-h { display: block; color: var(--muted); font-size: 10.5px; margin-top: 10px; }
          .bl-grid .bl-tier {
            grid-column: 1 / -1; width: 100%; margin: 6px 0 2px;
            font-size: 12px; font-weight: 600; color: var(--muted);
            display: flex; align-items: center; gap: 8px;
          }
          .bl-grid .bl-tier::before {
            content: ""; width: 3px; height: 13px; border-radius: 2px; background: var(--blue);
          }
          .bl-grid .bl-tier::after {
            content: ""; flex: 1; height: 1px; background: var(--rule);
          }
          .bl-pop .bl-ver {
            margin: 0 16px 4px; padding: 9px 12px; background: var(--bg-soft);
            border: 1px solid var(--rule); border-radius: 8px;
          }
          .bl-pop .bl-ver .bl-pop-row { margin-top: 6px; }
          .bl-pop .bl-ver .bl-pop-row:first-child { margin-top: 0; }
          .bl-pop .bl-chips { display: inline-flex; flex-wrap: wrap; gap: 5px; }
          .bl-pop .bl-chips code { margin: 0; }
          .bl-pop .bl-copyrow { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 11px; }
          .bl-pop .bl-copybtn {
            cursor: pointer; user-select: none; flex: 0 0 auto;
            display: inline-flex; align-items: center; gap: 5px;
            font-size: 11px; font-weight: 500; color: var(--blue-2);
            background: var(--blue-3); border: 1px solid var(--rule);
            padding: 4px 9px; border-radius: 6px; white-space: nowrap; line-height: 1.4;
            font-family: inherit; transition: color .15s, border-color .15s, background .15s;
          }
          .bl-pop .bl-copybtn::before { content: "⧉"; font-size: 11px; opacity: .85; }
          .bl-pop .bl-copybtn:hover { color: var(--blue); background: #fff; border-color: var(--blue); }
          .bl-pop .bl-copybtn:active { transform: translateY(1px); }
          .bl-pop .bl-jsonsrc { display: none; }
          .bl-pop .bl-midwrap { display: inline-flex; align-items: center; gap: 8px; flex-wrap: wrap; }
          .bl-pop .bl-midwrap > code { font-weight: 600; }
          .bl-pop .bl-copy-mini {
            flex: 0 0 auto; border: none; background: transparent; color: var(--blue-2);
            padding: 1px 5px; font-size: 11px; font-weight: 500;
          }
          .bl-pop .bl-copy-mini::before { font-size: 11px; opacity: .65; }
          .bl-pop .bl-copy-mini:hover { color: var(--blue); background: var(--blue-3); border-color: transparent; }
        

          
            
              
## 文本生成
查看更多 →
            
            

              
                
                  qwen3.8-max
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  qwen3.7-plus
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  qwen3.7-flash
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-flash`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-flash`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  deepseek-v4-pro阿里直供
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-pro`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-pro`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  deepseek-v4-flash阿里直供
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`deepseek-v4-flash`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`deepseek-v4-flash`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  kimi/kimi-k3三方直供
                          
                        
                华北2（北京）OpenAI 兼容模型 ID`kimi/kimi-k3`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗
                
                  glm-5.2阿里直供
                          
                        
                华北2（北京）新加坡德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`glm-5.2`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`glm-5.2`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`glm-5.2`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`glm-5.2`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  MiniMax-M3三方直供
                          
                        
                华北2（北京）OpenAI 兼容模型 ID`MiniMax/MiniMax-M3`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗
                
                  mimo-v2.5-pro三方直供
                          
                        
                华北2（北京）OpenAI 兼容模型 ID`xiaomi/mimo-v2.5-pro`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗
              
          

          
            
              
## 图像与视频

            
            
              
                
### 理解

                分析图片和视频内容，返回文本描述或结构化结果

查看更多 →
              
              
                
                  qwen3.8-max
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.8-max`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  qwen3.7-plus
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.7-plus`Base URL`https://dashscope-us.aliyuncs.com/api/v1`API Key获取↗
                
                  qwen3.5-omni-plus
                          
                          
                        
                华北2（北京）新加坡OpenAI 兼容模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗OpenAI 兼容模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗
                
                  kimi/kimi-k3三方直供
                          
                          
                        
                华北2（北京）OpenAI 兼容模型 ID`kimi/kimi-k3`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗
              
            
            
              
                
### 生成

                通过文本或图片生成图像与视频，支持编辑、参考与高分辨率输出

查看更多 →
              
              
                
                  qwen-image-3.0-pro
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen-image-3.0-pro`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation`API Key获取↗模型 ID`qwen-image-3.0-pro`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation`API Key获取↗
                
                  wan2.7-image-pro
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`wan2.7-image-pro`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/image-generation/generation`API Key获取↗模型 ID`wan2.7-image-pro`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/image-generation/generation`API Key获取↗
                
                  happyhorse-1.1-t2v
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）模型 ID`happyhorse-1.1-t2v`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-t2v`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-t2v`Request URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-t2v`Request URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-t2v`Request URL`https://dashscope-us.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗
                
                  happyhorse-1.1-i2v
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）模型 ID`happyhorse-1.1-i2v`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-i2v`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-i2v`Request URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-i2v`Request URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-i2v`Request URL`https://dashscope-us.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗
                
                  happyhorse-1.1-r2v
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）模型 ID`happyhorse-1.1-r2v`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-r2v`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-r2v`Request URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-r2v`Request URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.1-r2v`Request URL`https://dashscope-us.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗
                
                  happyhorse-1.0-video-edit
                          
                          
                        
                华北2（北京）新加坡日本（东京）德国（法兰克福）美国（弗吉尼亚）模型 ID`happyhorse-1.0-video-edit`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.0-video-edit`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.0-video-edit`Request URL`https://{WorkspaceId}.ap-northeast-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.0-video-edit`Request URL`https://{WorkspaceId}.eu-central-1.maas.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗模型 ID`happyhorse-1.0-video-edit`Request URL`https://dashscope-us.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis`API Key获取↗
              
            
          

          
            
              
## 3D模型生成

            
            
              
                文生3D模型或图生3D模型，构建三维资产

查看更多 →
              
              
                
                  Tripo/Tripo-H3.1三方直供
                          
                          
                        
                华北2（北京）模型 ID`Tripo/Tripo-H3.1`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/3d-generation`API Key获取↗
                
                  Tripo/Tripo-P1.0三方直供
                          
                          
                        
                华北2（北京）模型 ID`Tripo/Tripo-P1.0`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/video-generation/3d-generation`API Key获取↗
              
            
          

          
            
              
## 音频与语音

            
            
              
                
### 语音合成

                适用于有声阅读、语音播报、虚拟人等场景

查看更多 →
              
              
                
                  qwen-audio-3.0-tts-plus
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen-audio-3.0-tts-plus`Request URL`wss://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/inference`API Key获取↗模型 ID`qwen-audio-3.0-tts-plus`Request URL`wss://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api-ws/v1/inference`API Key获取↗
                
                  MiniMax/speech-2.8-hd三方直供
                          
                          
                        
                华北2（北京）模型 ID`MiniMax/speech-2.8-hd`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation`API Key获取↗
              
            
            
              
                
### 音乐生成

                根据提示词或歌词生成音乐

查看更多 →
              
              
                
                  fun-music-v1
                          
                          
                        
                华北2（北京）模型 ID`fun-music-v1`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/music/generation`API Key获取↗
              
            
            
              
                
### 语音识别

                专业 ASR 与大模型两种方案，按精度与灵活性选择

查看更多 →
              
              
                
                  qwen-audio-3.0-asr-flash-streaming
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen-audio-3.0-asr-flash-streaming`Request URL`wss://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/inference`API Key获取↗模型 ID`qwen-audio-3.0-asr-flash-streaming`Request URL`wss://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api-ws/v1/inference`API Key获取↗
                
                  qwen-audio-3.0-asr-flash-filetrans
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen-audio-3.0-asr-flash-filetrans`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/audio/asr/transcription`API Key获取↗模型 ID`qwen-audio-3.0-asr-flash-filetrans`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/asr/transcription`API Key获取↗
                
                  qwen3.5-omni-plus-realtime
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen3.5-omni-plus-realtime`Request URL`wss://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/realtime`API Key获取↗模型 ID`qwen3.5-omni-plus-realtime`Request URL`wss://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api-ws/v1/realtime`API Key获取↗
                
                  qwen3.5-omni-plus
                          
                          
                          
                        
                华北2（北京）新加坡OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗
              
            
            
              
                
### 语音转语音

                端到端语音对话，无需分别调用 ASR 和 TTS

查看更多 →
              
              
                
                  qwen-audio-3.0-realtime-plus
                          
                          
                          
                          
                        
                华北2（北京）模型 ID`qwen-audio-3.0-realtime-plus`Request URL`wss://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/realtime`API Key获取↗
                
                  qwen3.5-omni-plus
                          
                          
                          
                          
                        
                华北2（北京）新加坡OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗
              
            
          

          
            
              
## 全模态

            
            
              
                融合文本、图像、音频、视频等多种模态的理解与生成能力

查看更多 →
              
              
                
                  qwen3.5-omni-plus-realtime
                          
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`qwen3.5-omni-plus-realtime`Request URL`wss://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/realtime`API Key获取↗模型 ID`qwen3.5-omni-plus-realtime`Request URL`wss://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api-ws/v1/realtime`API Key获取↗
                
                  qwen3.5-omni-plus
                          
                          
                          
                          
                        
                华北2（北京）新加坡OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1`API Key获取↗OpenAI 兼容Anthropic 兼容DashScope模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`API Key获取↗模型 ID`qwen3.5-omni-plus`Base URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1`API Key获取↗
              
            
          

          
            
              
## 向量与重排序

            
            
              
                文本或图文向量化，配合重排序提升检索精度

查看更多 →
              
              
                
                  text-embedding-v4
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`text-embedding-v4`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/embeddings`API Key获取↗模型 ID`text-embedding-v4`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/embeddings`API Key获取↗
                
                  tongyi-embedding-vision-plus
                          
                          
                          
                        
                华北2（北京）新加坡模型 ID`tongyi-embedding-vision-plus`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding`API Key获取↗模型 ID`tongyi-embedding-vision-plus`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding`API Key获取↗
                
                  qwen3-rerank
                          
                        
                华北2（北京）新加坡模型 ID`qwen3-rerank`Request URL`https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank`API Key获取↗模型 ID`qwen3-rerank`Request URL`https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank`API Key获取↗
              
            
          

          
            
              
## 查看所有模型

            
            前往模型广场查看所有千问、三方、领域及历史版本模型。