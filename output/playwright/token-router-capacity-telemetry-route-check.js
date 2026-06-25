async (page) => {
  const baseUrl = "http://127.0.0.1:3000/token-router";
  const desktopScreenshot =
    "/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-capacity-telemetry.png";
  const mobileScreenshot =
    "/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-capacity-telemetry-mobile.png";

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
    id: 1,
    name: "GB10 Supply Lab",
    type: "self_hosted",
    status: 1,
    notes: "Playwright capacity telemetry supplier",
    created_time: 1782165800,
    updated_time: 1782165800,
  };

  const capacity = {
    id: 501,
    supplier_id: supplier.id,
    supply_node: "gb10-4t",
    model_name: "gpt-test",
    period_start: 1782162200,
    period_end: 1782169400,
    capacity_tokens: 1000,
    used_tokens: 300,
    headroom_tokens: 700,
    utilization_rate: 0.3,
    gpu_utilization_rate: 0.62,
    quality_score: 98.5,
    unit_cost_quota: 0.5,
    telemetry_source_type: "node_report",
    telemetry_source_ref: "process-gb10-4t-capacity-telemetry",
    telemetry_observed_at: 1782165800,
    last_telemetry_id: 901,
    status: 1,
    notes: "Playwright capacity telemetry snapshot",
    created_time: 1782165800,
    updated_time: 1782165800,
  };

  const requests = [];

  function ok(data) {
    return { success: true, message: "", data };
  }

  function pageData(items) {
    return ok({ page: 1, page_size: 100, total: items.length, items });
  }

  function assert(condition, message) {
    if (!condition) {
      throw new Error(message);
    }
  }

  function parseRequestUrl(rawUrl) {
    const pathWithSearch = rawUrl.replace(/^https?:\/\/[^/]+/, "");
    const queryIndex = pathWithSearch.indexOf("?");
    if (queryIndex === -1) {
      return { pathname: pathWithSearch, search: "" };
    }
    return {
      pathname: pathWithSearch.slice(0, queryIndex),
      search: pathWithSearch.slice(queryIndex),
    };
  }

  async function fulfillJson(route, body) {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  }

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
    const path = url.pathname.replace(/\/+$/, "");
    requests.push({
      method: request.method(),
      path,
      search: url.search,
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
    if (path === "/api/suppliers" && request.method() === "GET") {
      await fulfillJson(route, pageData([supplier]));
      return;
    }
    if (path === "/api/supply_capacities" && request.method() === "GET") {
      await fulfillJson(route, pageData([capacity]));
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
  `,
  });
  await page.getByRole("tab", { name: "Supply Capacity" }).click();
  await page.getByText("Supply Capacity Snapshots").waitFor();
const capacityTable = page.getByRole("table").filter({
  has: page.getByRole("columnheader", { name: "Telemetry" }),
});
await capacityTable.getByRole("columnheader", { name: "Quality / Cost" }).waitFor();
await capacityTable.getByText("GB10 Supply Lab").waitFor();
await capacityTable
  .getByRole("cell", { name: "gb10-4t", exact: true })
  .waitFor();
await capacityTable.getByText("gpt-test").waitFor();
await capacityTable.getByText("30%").waitFor();
await capacityTable.getByText("62%").waitFor();
  await capacityTable.getByText("node_report").waitFor();
  await capacityTable.getByText("process-gb10-4t-capacity-telemetry").waitFor();
  await capacityTable.getByText("#901").waitFor();
  await page.screenshot({ path: desktopScreenshot, fullPage: true });

  assert(
    requests.some((entry) => entry.method === "GET" && entry.path === "/api/supply_capacities"),
    "capacity GET was not observed",
  );

  await page.setViewportSize({ width: 390, height: 844 });
  await capacityTable.getByText("process-gb10-4t-capacity-telemetry").scrollIntoViewIfNeeded();
  await page.waitForTimeout(300);
  await page.screenshot({ path: mobileScreenshot, fullPage: false });

  console.log(
    JSON.stringify({
      verified: true,
      screenshots: [desktopScreenshot, mobileScreenshot],
      capacityGetCount: requests.filter(
        (entry) => entry.method === "GET" && entry.path === "/api/supply_capacities",
      ).length,
    }),
  );
}
