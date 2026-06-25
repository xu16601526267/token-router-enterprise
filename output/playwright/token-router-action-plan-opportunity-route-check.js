async (page) => {
const baseUrl = 'http://127.0.0.1:3000/token-router'
const desktopScreenshot =
  '/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-action-plan-opportunity.png'
const mobileScreenshot =
  '/Users/jiawei-macmini/projects/token-router/output/playwright/token-router-action-plan-opportunity-mobile.png'

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

const actionPlan = {
  id: 7101,
  supply_decision_id: 6101,
  decision_key: 'decision:self-hosted-cache:qwen3-72b:gold:42',
  supply_expansion_opportunity_id: 9001,
  opportunity_key: 'self-hosted-cache:qwen3-72b:gold:42',
  opportunity_type: 'self_hosted_cache',
  opportunity_priority: 'action',
  opportunity_cluster_key: 'high_cache_stable',
  opportunity_rank_score: 291,
  traffic_profile_id: 5101,
  slice_key: 'qwen3-72b|gold|42',
  model_name: 'qwen3-72b',
  sla_tier: 'gold',
  user_id: 42,
  period_start: 1782172800,
  period_end: 1782176400,
  decision_type: 'self_hosted_evaluate',
  track: 'self_hosted',
  action_type: 'evaluate_self_hosted_capacity',
  status: 'planned',
  recommended_capacity: 300,
  gap_tokens: 0,
  roi_score: 191,
  reason:
    'approved decision requires self-hosted capacity evaluation; register infrastructure only after operator approval',
  source_reviewed_at: 1782173000,
  source_reviewed_by: 1,
  generated_at: 1782173200,
  started_at: 0,
  completed_at: 0,
  cancelled_at: 0,
  status_updated_at: 0,
  status_updated_by: 0,
  operator_note: '',
  created_at: 1782173200,
  updated_at: 1782173200,
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
    path === '/api/supply_action_plans/generate' &&
    request.method() === 'POST'
  ) {
    await fulfillJson(route, ok([actionPlan]))
    return
  }
  if (path === '/api/supply_action_plans' && request.method() === 'GET') {
    const status = queryParam(url.search, 'status')
    const track = queryParam(url.search, 'track')
    const matchesStatus = !status || status === actionPlan.status
    const matchesTrack = !track || track === actionPlan.track
    await fulfillJson(
      route,
      pageData(matchesStatus && matchesTrack ? [actionPlan] : [])
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
await page.getByRole('tab', { name: 'Action Plans' }).click()
await page.getByText('Supply Action Plans').waitFor()
const actionPlansTable = page.getByRole('table').filter({
  has: page.getByRole('columnheader', { name: 'Opportunity' }),
})
await actionPlansTable.getByText('Self-hosted Cache').waitFor()
await actionPlansTable.getByText('High Cache Stable').waitFor()
await actionPlansTable.getByText('Rank Score: 291').waitFor()
await actionPlansTable.getByText('#9001').waitFor()
await actionPlansTable.getByText('qwen3-72b').waitFor()
await page.screenshot({ path: desktopScreenshot, fullPage: true })

const defaultGet = requests.find(
  (entry) =>
    entry.method === 'GET' &&
    entry.path === '/api/supply_action_plans' &&
    entry.search.includes('status=planned')
)
assert(defaultGet, 'default action plan GET was not observed')

await page.getByRole('button', { name: 'Self-hosted' }).click()
await page.waitForTimeout(250)
assert(
  requests.some(
    (entry) =>
      entry.method === 'GET' &&
      entry.path === '/api/supply_action_plans' &&
      entry.search.includes('track=self_hosted')
  ),
  'action plan track filter GET was not observed'
)

const generateRequest = page.waitForRequest((request) => {
  const url = parseRequestUrl(request.url())
  return (
    request.method() === 'POST' &&
    url.pathname === '/api/supply_action_plans/generate'
  )
})
await page.getByRole('button', { name: 'Generate Action Plans' }).click()
const generated = await generateRequest
const payload = JSON.parse(generated.postData() || '{}')
assert(payload.period_start > 0, 'generate payload missing period_start')
assert(payload.period_end >= payload.period_start, 'generate period is invalid')
assert(payload.track === 'self_hosted', 'generate payload missing track filter')
await page.getByText('Action plans generated').waitFor()

await page.setViewportSize({ width: 390, height: 844 })
await page
  .locator('[data-sonner-toast]')
  .waitFor({ state: 'detached', timeout: 6000 })
  .catch(() => {})
await actionPlansTable.getByText('Self-hosted Cache').scrollIntoViewIfNeeded()
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
