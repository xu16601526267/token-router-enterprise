async (page) => {
  const baseUrl = "http://127.0.0.1:4191/token-router/";
  const desktopScreenshot =
    "output/playwright/adr0078-supplier-posture-boost-recommendations.png";
  const mobileScreenshot =
    "output/playwright/adr0078-supplier-posture-boost-recommendations-mobile.png";
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
    id: 5,
    name: "gb10-4t-strong-supplier",
    type: "third_party",
    status: 1,
    notes: "Strong scorecard posture supplier",
    created_time: periodStart,
    updated_time: periodStart,
  };

  const boostRecommendation = {
    id: 78,
    supplier_id: supplier.id,
    supplier_scorecard_id: 88,
    period_start: periodStart,
    period_end: periodEnd,
    score: 94,
    grade: "A",
    recommended_action: "boost",
    reason:
      "scorecard grade A score 94.000 with no open supplier posture insights meets boost review threshold",
    quality_insight_count: 0,
    capacity_insight_count: 0,
    action_insight_count: 0,
    total_requests: 1800,
    success_rate: 0.99,
    avg_latency_ms: 180,
    supply_headroom_tokens: 88000,
    avg_supply_quality_score: 98,
    supplier_status_current: 1,
    supplier_status_before: 1,
    supplier_status_after: 1,
    status: "applied",
    reviewed_at: periodStart + 300,
    reviewed_by: 1,
    review_note: "approved posture boost recommendation from dashboard",
    applied_at: periodStart + 600,
    applied_by: 1,
    applied_note: "applied posture boost recommendation from dashboard",
    created_at: periodStart + 60,
    updated_at: periodStart + 60,
  };

  const routePreferences = [
    {
      id: 9200,
      supplier_id: supplier.id,
      source_posture_recommendation_id: boostRecommendation.id,
      status: "active",
      weight_percent: 150,
      reason: "supplier_posture_recommendation #78 boost: grade=A score=94.000",
      effective_from: periodStart + 600,
      effective_to: 0,
      activated_at: periodStart + 600,
      activated_by: 1,
      disabled_at: 0,
      disabled_by: 0,
      operator_note: "applied posture boost recommendation from dashboard",
      created_at: periodStart + 600,
      updated_at: periodStart + 600,
    },
  ];

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
    requests.push({
      method,
      path,
      search: url.search,
      params: url.params,
      postData: request.postData(),
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
      const action = url.params.recommended_action || "";
      const items = action === "" || action === "boost" ? [boostRecommendation] : [];
      await fulfillJson(route, pageData(items));
      return;
    }
    if (path === "/api/supplier_route_preferences" && method === "GET") {
      await fulfillJson(route, pageData(routePreferences));
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
  await page.getByText("Supplier Posture Recommendations", { exact: true }).waitFor();
  await page.getByText("Runtime supplier posture recommendations").waitFor();
  await page.getByText("Boost", { exact: true }).first().waitFor();
  await page.getByText("Route Preference Active").first().waitFor();
  await page.getByText("150%").first().waitFor();

  const actionGroup = page.getByLabel("Posture Action");
  await actionGroup.getByText("Boost", { exact: true }).click();
  await waitForRequest(
    page,
    (entry) =>
      entry.method === "GET" &&
      entry.path === "/api/supplier_posture_recommendations" &&
      entry.params.recommended_action === "boost",
    "boost action filter did not request recommended_action=boost",
  );
  await page.getByText("gb10-4t-strong-supplier").first().waitFor();
  await page.getByText("scorecard grade A score 94.000").waitFor();

  await page.evaluate(() => {
    for (const element of document.querySelectorAll("*")) {
      element.scrollLeft = 0;
    }
  });
  await page
    .getByText("Supplier Posture Recommendations", { exact: true })
    .evaluate((element) => {
      element.scrollIntoView({ block: "center", inline: "nearest" });
    });
  await page.waitForTimeout(300);
  await page.screenshot({ path: desktopScreenshot, fullPage: false });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByText("Route Preference Active").first().scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await page.screenshot({ path: mobileScreenshot, fullPage: false });

  const boostFilteredGets = requests.filter(
    (entry) =>
      entry.method === "GET" &&
      entry.path === "/api/supplier_posture_recommendations" &&
      entry.params.recommended_action === "boost",
  );
  const routePreferenceGets = requests.filter(
    (entry) => entry.method === "GET" && entry.path === "/api/supplier_route_preferences",
  );

  assert(boostFilteredGets.length >= 1, "boost filter GET was not observed");
  assert(routePreferenceGets.length >= 1, "active route preference GET was not observed");
  assert(
    routePreferenceGets.every((entry) => entry.params.status === "active"),
    "route preference GET did not request active preferences",
  );

  console.log(
    JSON.stringify({
      verified: true,
      screenshots: [desktopScreenshot, mobileScreenshot],
      boostFilteredGetCount: boostFilteredGets.length,
      routePreferenceGetCount: routePreferenceGets.length,
    }),
  );
}
