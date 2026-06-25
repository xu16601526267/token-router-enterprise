async (page) => {
  const baseUrl = "http://127.0.0.1:4190/token-router/";
  const desktopScreenshot = "output/playwright/adr0077-bounded-supplier-route-preference-boost.png";
  const mobileScreenshot =
    "output/playwright/adr0077-bounded-supplier-route-preference-boost-mobile.png";
  const periodStart = 1782162180;
  const periodEnd = 1782169380;
  const requests = [];

  const adminUser = {
    id: 1,
    username: "playwright-admin",
    display_name: "Playwright Admin",
    role: 100,
    status: 1,
    group: "default",
    quota: 1000000,
    used_quota: 0,
    request_count: 0,
  };

  const supplier = {
    id: 3,
    name: "gb10-4t-self-operated",
    type: "self_operated",
    status: 1,
    notes: "GB10 4T posture supplier",
    created_time: periodStart,
    updated_time: periodStart,
  };

  const postureRecommendation = {
    id: 72,
    supplier_id: supplier.id,
    supplier_scorecard_id: 42,
    period_start: periodStart,
    period_end: periodEnd,
    score: 65,
    grade: "C",
    recommended_action: "downgrade",
    reason: "watch quality trend but keep supplier enabled",
    quality_insight_count: 2,
    capacity_insight_count: 1,
    action_insight_count: 1,
    total_requests: 1200,
    success_rate: 0.82,
    avg_latency_ms: 1880,
    supply_headroom_tokens: 12000,
    avg_supply_quality_score: 0.58,
    supplier_status_current: 1,
    supplier_status_before: 0,
    supplier_status_after: 0,
    status: "applied",
    reviewed_at: periodStart + 300,
    reviewed_by: 1,
    review_note: "approved supplier posture recommendation from dashboard",
    applied_at: periodStart + 600,
    applied_by: 1,
    applied_note: "applied supplier posture recommendation from dashboard",
    created_at: periodStart + 60,
    updated_at: periodStart + 60,
  };

  let routePreferences = [];

  const ok = (data) => ({ success: true, message: "", data });
  const pageData = (items) => ok({ page: 1, page_size: 100, total: items.length, items });
  const assert = (condition, message) => {
    if (!condition) throw new Error(message);
  };
  const waitForRequest = async (page, predicate, message) => {
    for (let i = 0; i < 60; i += 1) {
      if (requests.some(predicate)) return;
      await page.waitForTimeout(100);
    }
    throw new Error(message);
  };
  const parseSearch = (search) => {
    const params = {};
    if (!search) return params;
    for (const pair of search.split("&")) {
      if (!pair) continue;
      const equalsIndex = pair.indexOf("=");
      const rawKey = equalsIndex === -1 ? pair : pair.slice(0, equalsIndex);
      const rawValue = equalsIndex === -1 ? "" : pair.slice(equalsIndex + 1);
      params[decodeURIComponent(rawKey)] = decodeURIComponent(rawValue);
    }
    return params;
  };
  const parseRequestUrl = (rawUrl) => {
    const normalized = rawUrl.replace(/^https?:\/\/[^/]+/, "");
    const [pathname, search = ""] = normalized.split("?");
    return {
      pathname: pathname.replace(/\/+$/, ""),
      params: parseSearch(search),
      search,
    };
  };
  const fulfillJson = async (route, body) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  };

  await page.context().clearCookies();
  await page.addInitScript((user) => {
    window.localStorage.setItem("user", JSON.stringify(user));
    window.localStorage.setItem("uid", String(user.id));
    window.localStorage.setItem("setup_status_checked", "true");
    window.localStorage.setItem("i18nextLng", "en");
    window.localStorage.setItem("language", "en");
  }, adminUser);

  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = parseRequestUrl(request.url());
    const path = url.pathname;
    const method = request.method();
    const postData = request.postData();
    requests.push({
      method,
      path,
      search: url.search,
      params: url.params,
      postData,
    });

    if (path === "/api/user/self") {
      await fulfillJson(route, ok(adminUser));
      return;
    }
    if (path === "/api/status") {
      await fulfillJson(route, ok({ version: "playwright", footer_html: "" }));
      return;
    }
    if (path === "/api/setup") {
      await fulfillJson(route, ok({ status: true, root_init: true, database_type: "sqlite" }));
      return;
    }
    if (path === "/api/notice") {
      await fulfillJson(route, ok(""));
      return;
    }
    if (path === "/api/suppliers" && method === "GET") {
      await fulfillJson(route, pageData([supplier]));
      return;
    }
    if (path === "/api/supplier_posture_recommendations" && method === "GET") {
      await fulfillJson(route, pageData([postureRecommendation]));
      return;
    }
    if (path === "/api/supplier_route_preferences" && method === "GET") {
      await fulfillJson(route, pageData(routePreferences));
      return;
    }
    if (path === "/api/supplier_route_preferences/activate" && method === "POST") {
      const input = JSON.parse(postData || "{}");
      assert(input.supplier_id === supplier.id, "activate supplier_id drifted");
      assert(input.weight_percent === 150, "activate boost weight_percent drifted");
      assert(
        input.reason === "operator manual route preference boost in playwright",
        "activate reason drifted",
      );
      routePreferences = [
        {
          id: 9100,
          supplier_id: supplier.id,
          source_posture_recommendation_id: 0,
          status: "active",
          weight_percent: input.weight_percent,
          reason: input.reason,
          effective_from: periodStart + 900,
          effective_to: 0,
          activated_at: periodStart + 900,
          activated_by: 1,
          disabled_at: 0,
          disabled_by: 0,
          operator_note: input.operator_note,
          created_at: periodStart + 900,
          updated_at: periodStart + 900,
        },
      ];
      await fulfillJson(route, ok(routePreferences[0]));
      return;
    }
    if (path === `/api/supplier_route_preferences/${supplier.id}/disable` && method === "POST") {
      const input = JSON.parse(postData || "{}");
      assert(
        input.operator_note.includes("disabled supplier route preference"),
        "disable operator_note drifted",
      );
      const disabled = {
        ...routePreferences[0],
        status: "disabled",
        weight_percent: 100,
        effective_to: periodStart + 1200,
        disabled_at: periodStart + 1200,
        disabled_by: 1,
        operator_note: input.operator_note,
      };
      routePreferences = [];
      await fulfillJson(route, ok(disabled));
      return;
    }
    if (path === "/api/reports/margin_summary" || path === "/api/reports/quality_summary") {
      await fulfillJson(route, ok([]));
      return;
    }

    await fulfillJson(route, pageData([]));
  });

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto(baseUrl, { waitUntil: "domcontentloaded" });
  await page.addStyleTag({
    content: `
      [aria-label="Open Tanstack query devtools"],
      [aria-label="Open TanStack Router Devtools"] {
        display: none !important;
      }
      [data-sonner-toaster] {
        display: none !important;
      }
    `,
  });
  await page.getByLabel("Period Start").fill("2026-06-23T05:03");
  await page.getByLabel("Period End").fill("2026-06-23T07:03");
  await page.getByRole("tab", { name: "Posture" }).click();
  await page.getByText("Active Route Preferences", { exact: true }).last().waitFor();
  await page.getByText("No active supplier route preferences.").first().waitFor();

  const setRoutePreferenceButton = page.getByRole("button", {
    name: "Set Route Preference",
  });
  await setRoutePreferenceButton.click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Supplier").selectOption(String(supplier.id));
  const routeWeightInput = dialog.getByLabel("Route Weight Percent");
  assert(
    (await routeWeightInput.getAttribute("max")) === "200",
    "route weight input max did not expose bounded boost cap",
  );
  await routeWeightInput.fill("150");
  await dialog.getByLabel("Reason").fill("operator manual route preference boost in playwright");
  await dialog.getByLabel("Operator Note").fill("playwright manual watch");
  await dialog.getByRole("button", { name: "Set Route Preference" }).click();

  await waitForRequest(
    page,
    (entry) => entry.method === "POST" && entry.path === "/api/supplier_route_preferences/activate",
    "activate POST was not observed after submitting route preference",
  );
  await page.getByText("Manual", { exact: true }).first().waitFor();
  await page.getByText("150%").first().waitFor();
  await page.getByText("operator manual route preference boost in playwright").first().waitFor();

  await page.evaluate(() => {
    for (const element of document.querySelectorAll("*")) {
      element.scrollLeft = 0;
    }
  });
  await page
    .getByText("Active Route Preferences", { exact: true })
    .last()
    .evaluate((element) => {
      element.scrollIntoView({ block: "center", inline: "nearest" });
    });
  await page.waitForTimeout(300);
  await page.screenshot({ path: desktopScreenshot, fullPage: false });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByText("Active Route Preferences", { exact: true }).last().scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await page.screenshot({ path: mobileScreenshot, fullPage: false });

  await page.setViewportSize({ width: 1440, height: 980 });
  await page
    .getByRole("row", { name: /Manual/ })
    .getByRole("button", { name: "Disable", exact: true })
    .click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Disable", exact: true }).click();
  await waitForRequest(
    page,
    (entry) =>
      entry.method === "POST" &&
      entry.path === `/api/supplier_route_preferences/${supplier.id}/disable`,
    "disable POST was not observed after confirming disable",
  );
  await page.getByText("No active supplier route preferences.").first().waitFor();

  const routePreferenceGets = requests.filter(
    (entry) => entry.method === "GET" && entry.path === "/api/supplier_route_preferences",
  );
  const activatePosts = requests.filter(
    (entry) => entry.method === "POST" && entry.path === "/api/supplier_route_preferences/activate",
  );
  const disablePosts = requests.filter(
    (entry) =>
      entry.method === "POST" &&
      entry.path === `/api/supplier_route_preferences/${supplier.id}/disable`,
  );

  assert(activatePosts.length === 1, "activate POST was not observed once");
  assert(disablePosts.length === 1, "disable POST was not observed once");
  assert(
    routePreferenceGets.length >= 3,
    "route preference GET did not refetch after activate and disable",
  );
  assert(
    routePreferenceGets.every((entry) => entry.params.status === "active"),
    "route preference GET did not request active preferences",
  );

  console.log(
    JSON.stringify({
      verified: true,
      screenshots: [desktopScreenshot, mobileScreenshot],
      routePreferenceGetCount: routePreferenceGets.length,
      activatePostCount: activatePosts.length,
      disablePostCount: disablePosts.length,
    }),
  );
}
