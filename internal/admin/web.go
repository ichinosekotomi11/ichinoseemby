package admin

const adminDashboardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ichinose Emby Admin</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f5f7fb;
      --panel: #ffffff;
      --line: #d9e0ea;
      --text: #172033;
      --muted: #667085;
      --blue: #2563eb;
      --green: #0f9f6e;
      --red: #dc2626;
      --amber: #d97706;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    header {
      position: sticky;
      top: 0;
      z-index: 2;
      background: rgba(255,255,255,.92);
      border-bottom: 1px solid var(--line);
      backdrop-filter: blur(12px);
    }
    .bar, main { width: min(1180px, calc(100vw - 32px)); margin: 0 auto; }
    .bar { height: 64px; display: flex; align-items: center; justify-content: space-between; gap: 16px; }
    h1 { margin: 0; font-size: 20px; }
    h2 { margin: 0 0 14px; font-size: 16px; }
    main { padding: 24px 0 40px; display: grid; gap: 18px; }
    .grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
    .columns { display: grid; grid-template-columns: 1.05fr .95fr; gap: 18px; align-items: start; }
    section, .stat {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      box-shadow: 0 10px 22px rgba(15,23,42,.04);
    }
    section { padding: 18px; }
    .stat { padding: 16px; }
    .stat strong { display: block; font-size: 26px; line-height: 1.1; }
    .stat span, .muted { color: var(--muted); }
    .login {
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
    }
    .login section { width: min(420px, 100%); }
    label { display: block; margin: 10px 0 6px; color: var(--muted); font-weight: 600; }
    input, select, textarea {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 10px 11px;
      background: #fff;
      color: var(--text);
      font: inherit;
    }
    textarea { min-height: 260px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
    button {
      border: 0;
      border-radius: 6px;
      padding: 10px 13px;
      background: var(--blue);
      color: #fff;
      font-weight: 700;
      cursor: pointer;
    }
    button.secondary { background: #e8eef8; color: #1d2b47; }
    button.danger { background: var(--red); }
    .actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
    .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; }
    th { color: var(--muted); font-size: 12px; font-weight: 700; }
    .badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 28px;
      height: 24px;
      border-radius: 999px;
      font-weight: 800;
      color: #fff;
    }
    .level-A { background: var(--amber); }
    .level-B { background: var(--green); }
    .level-C { background: #64748b; }
    .toast {
      min-height: 22px;
      color: var(--muted);
      font-weight: 600;
    }
    .error { color: var(--red); }
    .ok { color: var(--green); }
    .hidden { display: none; }
    @media (max-width: 900px) {
      .grid, .columns, .form-grid { grid-template-columns: 1fr; }
      .bar { height: auto; padding: 14px 0; align-items: flex-start; flex-direction: column; }
      table { display: block; overflow-x: auto; white-space: nowrap; }
    }
  </style>
</head>
<body>
  <div id="login" class="login">
    <section>
      <h1>Ichinose Emby Admin</h1>
      <p class="muted">输入部署时设置的 EDEN_ADMIN_TOKEN。</p>
      <label for="token">管理密钥</label>
      <input id="token" type="password" autocomplete="current-password" placeholder="例如 w123456">
      <div style="height:14px"></div>
      <div class="actions">
        <button onclick="login()">登录</button>
        <button class="secondary" onclick="fillSaved()">读取已保存密钥</button>
      </div>
      <p id="loginMsg" class="toast"></p>
    </section>
  </div>

  <div id="app" class="hidden">
    <header>
      <div class="bar">
        <div>
          <h1>Ichinose Emby Admin</h1>
          <div class="muted">用户等级、金币、开放注册和功能权限管理</div>
        </div>
        <div class="actions">
          <button class="secondary" onclick="refreshAll()">刷新</button>
          <button class="danger" onclick="logout()">退出</button>
        </div>
      </div>
    </header>
    <main>
      <div class="grid">
        <div class="stat"><span>总用户</span><strong id="totalUsers">0</strong></div>
        <div class="stat"><span>A 级白名单</span><strong id="levelA">0</strong></div>
        <div class="stat"><span>B 级可用</span><strong id="levelB">0</strong></div>
        <div class="stat"><span>C 级过期/游客</span><strong id="levelC">0</strong></div>
      </div>

      <section>
        <h2>新增用户</h2>
        <div class="form-grid">
          <div><label>用户名</label><input id="newUsername" placeholder="username"></div>
          <div><label>邮箱</label><input id="newEmail" placeholder="optional@example.com"></div>
          <div><label>等级</label><select id="newLevel"><option>A</option><option>B</option><option selected>C</option></select></div>
          <div><label>初始金币</label><input id="newCoins" type="number" value="0"></div>
          <div><label>本地密码</label><input id="newPassword" type="password" placeholder="可留空"></div>
          <div style="display:flex; align-items:end"><button onclick="createUser()">创建用户</button></div>
        </div>
      </section>

      <div class="columns">
        <section>
          <h2>用户管理</h2>
          <table>
            <thead>
              <tr><th>用户</th><th>等级</th><th>金币</th><th>创建时间</th><th>操作</th></tr>
            </thead>
            <tbody id="usersBody"></tbody>
          </table>
        </section>
        <section>
          <h2>功能权限矩阵</h2>
          <p class="muted">直接编辑 JSON 后保存。allowed_levels 控制 A/B/C 可用范围，coin_cost 和倍率控制消耗。</p>
          <textarea id="featuresText"></textarea>
          <div style="height:10px"></div>
          <div class="actions">
            <button onclick="saveFeatures()">保存功能配置</button>
            <button class="secondary" onclick="loadFeatures()">恢复服务器配置</button>
          </div>
        </section>
      </div>
      <p id="msg" class="toast"></p>
    </main>
  </div>

  <script>
    const $ = (id) => document.getElementById(id);
    let adminToken = localStorage.getItem("ichinose_admin_token") || "";
    let users = [];

    function headers() {
      return {"Content-Type": "application/json", "X-Admin-Token": adminToken};
    }

    function setMsg(text, ok = true) {
      $("msg").textContent = text;
      $("msg").className = "toast " + (ok ? "ok" : "error");
    }

    function setLoginMsg(text, ok = true) {
      $("loginMsg").textContent = text;
      $("loginMsg").className = "toast " + (ok ? "ok" : "error");
    }

    function fillSaved() {
      $("token").value = adminToken;
      setLoginMsg(adminToken ? "已填入本地保存的密钥" : "本地没有保存密钥", Boolean(adminToken));
    }

    async function api(path, options = {}) {
      const res = await fetch(path, {...options, headers: {...headers(), ...(options.headers || {})}});
      const text = await res.text();
      let body = null;
      try { body = text ? JSON.parse(text) : null; } catch { body = text; }
      if (!res.ok) throw new Error((body && body.error) || text || "请求失败");
      return body;
    }

    async function login() {
      adminToken = $("token").value.trim();
      if (!adminToken) return setLoginMsg("请输入管理密钥", false);
      try {
        await api("/api/admin/security-design");
        localStorage.setItem("ichinose_admin_token", adminToken);
        $("login").classList.add("hidden");
        $("app").classList.remove("hidden");
        await refreshAll();
      } catch (err) {
        setLoginMsg(err.message, false);
      }
    }

    function logout() {
      localStorage.removeItem("ichinose_admin_token");
      adminToken = "";
      $("app").classList.add("hidden");
      $("login").classList.remove("hidden");
      $("token").value = "";
    }

    async function refreshAll() {
      await Promise.all([loadUsers(), loadFeatures()]);
      setMsg("已刷新");
    }

    async function loadUsers() {
      users = await api("/api/admin/users");
      renderUsers();
    }

    async function loadFeatures() {
      const features = await api("/api/admin/features");
      $("featuresText").value = JSON.stringify(features, null, 2);
    }

    function renderUsers() {
      const counts = {A: 0, B: 0, C: 0};
      for (const user of users) counts[user.level] = (counts[user.level] || 0) + 1;
      $("totalUsers").textContent = users.length;
      $("levelA").textContent = counts.A || 0;
      $("levelB").textContent = counts.B || 0;
      $("levelC").textContent = counts.C || 0;
      const body = $("usersBody");
      body.innerHTML = "";
      for (const user of users) {
        const tr = document.createElement("tr");
        const createdAt = user.created_at ? new Date(user.created_at).toLocaleString() : "-";
        tr.innerHTML =
          "<td><strong></strong><div class=\"muted\"></div></td>" +
          "<td><span class=\"badge\"></span></td>" +
          "<td></td>" +
          "<td></td>" +
          "<td><div class=\"actions\"></div></td>";
        tr.querySelector("strong").textContent = user.username;
        tr.querySelector(".muted").textContent = user.email || user.id;
        const badge = tr.querySelector(".badge");
        badge.classList.add("level-" + user.level);
        badge.textContent = user.level;
        tr.children[2].textContent = user.coins;
        tr.children[3].textContent = createdAt;

        const actions = tr.querySelector(".actions");
        const select = document.createElement("select");
        select.id = "level-" + user.id;
        for (const level of ["A", "B", "C"]) {
          const option = document.createElement("option");
          option.value = level;
          option.textContent = level;
          option.selected = user.level === level;
          select.appendChild(option);
        }
        const levelButton = document.createElement("button");
        levelButton.className = "secondary";
        levelButton.textContent = "改等级";
        levelButton.onclick = () => updateLevel(user.id);
        const coinInput = document.createElement("input");
        coinInput.id = "coin-" + user.id;
        coinInput.type = "number";
        coinInput.value = "10";
        coinInput.style.width = "88px";
        const coinButton = document.createElement("button");
        coinButton.className = "secondary";
        coinButton.textContent = "调金币";
        coinButton.onclick = () => changeCoins(user.id);
        actions.append(select, levelButton, coinInput, coinButton);
        body.appendChild(tr);
      }
    }

    async function createUser() {
      try {
        await api("/api/admin/users", {
          method: "POST",
          body: JSON.stringify({
            username: $("newUsername").value.trim(),
            email: $("newEmail").value.trim(),
            level: $("newLevel").value,
            coins: Number($("newCoins").value || 0),
            password: $("newPassword").value
          })
        });
        $("newUsername").value = "";
        $("newEmail").value = "";
        $("newPassword").value = "";
        await loadUsers();
        setMsg("用户已创建");
      } catch (err) {
        setMsg(err.message, false);
      }
    }

    async function updateLevel(id) {
      try {
        await api("/api/admin/users/" + id, {
          method: "PATCH",
          body: JSON.stringify({level: $("level-" + id).value})
        });
        await loadUsers();
        setMsg("等级已更新");
      } catch (err) {
        setMsg(err.message, false);
      }
    }

    async function changeCoins(id) {
      try {
        await api("/api/admin/users/" + id, {
          method: "PATCH",
          body: JSON.stringify({coin_delta: Number($("coin-" + id).value || 0), reason: "后台面板调整金币"})
        });
        await loadUsers();
        setMsg("金币已调整");
      } catch (err) {
        setMsg(err.message, false);
      }
    }

    async function saveFeatures() {
      try {
        const parsed = JSON.parse($("featuresText").value);
        await api("/api/admin/features", {method: "PUT", body: JSON.stringify(parsed)});
        setMsg("功能配置已保存");
      } catch (err) {
        setMsg(err.message, false);
      }
    }

    if (adminToken) {
      $("token").value = adminToken;
      login();
    }
  </script>
</body>
</html>`
