async (page) => {
const baseUrl = 'http://127.0.0.1:3000/token-router'
const desktopScreenshot =
  '/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-opportunities.png'
const mobileScreenshot =
  '/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-opportunities-mobile.png'

const adminUser = {
  id: 1,
  username: 'playwright-admin',
  display_name: 'Playwright Admin',
  role: 100,
  status: 1,
  group: 'default',
  quota: 1000000,
  used_quota: 0,
  request_count: 0,
}

const opportunity = {
  id: 9001,
  opportunity_key: 'self-hosted-cache:qwen3-72b:gold:42',
  supply_decision_id: 1,
  traffic_profile_id: 1,
  traffic_forecast_id: 1,
  decision_source: 'forecast',
  decision_status: 'approved',
  forecast_target_period_start: 1782176400,
  forecast_target_period_end: 1782180000,
  forecast_confidence: 0.82,
  forecast_method: 'moving_average:v1',
  model_name: 'qwen3-72b',
  sla_tier: 'gold',
  user_id: 42,
  track: 'self_hosted',
  decision_type: 'self_hosted_evaluate',
  opportunity_type: 'self_hosted_cache',
  priority: 'action',
  cluster_key: 'high_cache_stable',
  demand_tokens: 2000,
  peak_tokens: 880,
  supply_headroom_tokens: 120,
  gap_tokens: 760,
  recommended_capacity: 300,
  total_cached_tokens: 1000,
  cache_hit_rate: 0.5,
  sla_attainment_rate: 1,
  gross_profit_quota: 18.4,
  roi_score: 42,
  peak_ratio: 0.44,
  unique_sessions: 16,
  locality_score: 0.5,
  stability_score: 1,
  headroom_risk_score: 0,
  rank_score: 291,
  reason: 'High cache locality and stable approved self-hosted decision',
  created_time: 1782176500,
  updated_time: 1782176500,
}

const requests = []

function ok(data) {
  return { success: true, message: '', data }
}

function pageData(items) {
  return ok({ page: 1, page_size: 100, total: items.length, items })
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

function parseRequestUrl(rawUrl) {
  const pathWithSearch = rawUrl.replace(/^https?:\/\/[^/]+/, '')
  const queryIndex = pathWithSearch.indexOf('?')
  if (queryIndex === -1) {
    return { pathname: pathWithSearch, search: '' }
  }
  return {
    pathname: pathWithSearch.slice(0, queryIndex),
    search: pathWithSearch.slice(queryIndex),
  }
}

function queryParam(search, name) {
  const query = search.startsWith('?') ? search.slice(1) : search
  for (const part of query.split('&')) {
    if (!part) continue
    const [rawKey, rawValue = ''] = part.split('=')
    if (decodeURIComponent(rawKey) === name) {
      return decodeURIComponent(rawValue.replace(/\+/g, ' '))
    }
  }
  return null
}

async function fulfillJson(route, body) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

await page.context().clearCookies()
await page.addInitScript((user) => {
  window.localStorage.setItem('user', JSON.stringify(user))
  window.localStorage.setItem('uid', String(user.id))
  window.localStorage.setItem('setup_status_checked', 'true')
  window.localStorage.setItem('i18nextLng', 'en')
  window.localStorage.setItem('language', 'en')
}, adminUser)

await page.route('**/api/**', async (route) => {
  const request = route.request()
  const url = parseRequestUrl(request.url())
  const path = url.pathname.replace(/\/+$/, '')
  requests.push({
    method: request.method(),
    path,
    search: url.search,
    postData: request.postData(),
  })

  if (path === '/api/user/self') {
    await fulfillJson(route, ok(adminUser))
    return
  }
  if (path === '/api/status') {
    await fulfillJson(route, ok({ version: 'playwright', footer_html: '' }))
    return
  }
  if (path === '/api/setup') {
    await fulfillJson(
      route,
      ok({ status: true, root_init: true, database_type: 'sqlite' })
    )
    return
  }
  if (path === '/api/notice') {
    await fulfillJson(route, ok(''))
    return
  }
  if (
    path === '/api/reports/margin_summary' ||
    path === '/api/reports/quality_summary'
  ) {
    await fulfillJson(route, ok([]))
    return
  }
  if (
    path === '/api/supply_expansion_opportunities/generate' &&
    request.method() === 'POST'
  ) {
    await fulfillJson(route, ok([opportunity]))
    return
  }
  if (
    path === '/api/supply_expansion_opportunities' &&
    request.method() === 'GET'
  ) {
    const type = queryParam(url.search, 'opportunity_type')
    const priority = queryParam(url.search, 'priority')
    const matchesType = !type || type === opportunity.opportunity_type
    const matchesPriority = !priority || priority === opportunity.priority
    await fulfillJson(
      route,
      pageData(matchesType && matchesPriority ? [opportunity] : [])
    )
    return
  }

  await fulfillJson(route, pageData([]))
})

await page.setViewportSize({ width: 1440, height: 980 })
await page.goto(baseUrl, { waitUntil: 'domcontentloaded' })
await page.addStyleTag({
  content: `
    [aria-label="Open Tanstack query devtools"],
    [aria-label="Open TanStack Router Devtools"] {
      display: none !important;
    }
  `,
})
await page.getByRole('tab', { name: 'Opportunities' }).click()
await page.getByText('Supply Opportunities').waitFor()
const opportunitiesTable = page.getByRole('table').filter({
  has: page.getByRole('columnheader', { name: 'Opportunity' }),
})
await opportunitiesTable.getByText('Self-hosted Cache').waitFor()
await opportunitiesTable.getByText('High Cache Stable').waitFor()
await opportunitiesTable.getByText('High cache locality').waitFor()
await page.screenshot({ path: desktopScreenshot, fullPage: true })

const defaultGet = requests.find(
  (entry) =>
    entry.method === 'GET' &&
    entry.path === '/api/supply_expansion_opportunities' &&
    !entry.search.includes('opportunity_type=') &&
    !entry.search.includes('priority=')
)
assert(defaultGet, 'default opportunities GET was not observed')

await page.getByRole('button', { name: 'Self-hosted Cache' }).click()
await page.waitForTimeout(250)
assert(
  requests.some(
    (entry) =>
      entry.method === 'GET' &&
      entry.path === '/api/supply_expansion_opportunities' &&
      entry.search.includes('opportunity_type=self_hosted_cache')
  ),
  'opportunity type filter GET was not observed'
)

await page.getByRole('button', { name: /^Action$/ }).click()
await page.waitForTimeout(250)
assert(
  requests.some(
    (entry) =>
      entry.method === 'GET' &&
      entry.path === '/api/supply_expansion_opportunities' &&
      entry.search.includes('priority=action')
  ),
  'opportunity priority filter GET was not observed'
)

const generateRequest = page.waitForRequest((request) => {
  const url = parseRequestUrl(request.url())
  return (
    request.method() === 'POST' &&
    url.pathname === '/api/supply_expansion_opportunities/generate'
  )
})
await page.getByRole('button', { name: 'Generate Opportunities' }).click()
const generated = await generateRequest
const payload = JSON.parse(generated.postData() || '{}')
assert(payload.period_start > 0, 'generate payload missing period_start')
assert(payload.period_end >= payload.period_start, 'generate period is invalid')
await page.getByText('Supply opportunities generated').waitFor()

await page.setViewportSize({ width: 390, height: 844 })
await page
  .locator('[data-sonner-toast]')
  .waitFor({ state: 'detached', timeout: 6000 })
  .catch(() => {})
await opportunitiesTable.getByText('qwen3-72b').scrollIntoViewIfNeeded()
await page.waitForTimeout(300)
await page.screenshot({ path: mobileScreenshot, fullPage: false })

console.log(
  JSON.stringify(
    {
      verified: true,
      defaultGet: defaultGet.search,
      generatePayload: payload,
      screenshots: [desktopScreenshot, mobileScreenshot],
    },
    null,
    2
  )
)
}
