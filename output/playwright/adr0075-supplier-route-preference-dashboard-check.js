async (page) => {
  const baseUrl = 'http://127.0.0.1:4190/token-router/'
  const desktopScreenshot =
    'output/playwright/adr0075-supplier-route-preference-dashboard.png'
  const mobileScreenshot =
    'output/playwright/adr0075-supplier-route-preference-dashboard-mobile.png'
  const periodStart = 1782162180
  const periodEnd = 1782169380
  const requests = []

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

  const supplier = {
    id: 3,
    name: 'gb10-4t-self-operated',
    type: 'self_operated',
    status: 1,
    notes: 'GB10 4T posture supplier',
    created_time: periodStart,
    updated_time: periodStart,
  }

  const draftRecommendation = {
    id: 71,
    supplier_id: supplier.id,
    supplier_scorecard_id: 41,
    period_start: periodStart,
    period_end: periodEnd,
    score: 58,
    grade: 'D',
    recommended_action: 'disable',
    reason: 'quality and capacity incidents exceed posture threshold',
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
    status: 'draft',
    reviewed_at: 0,
    reviewed_by: 0,
    review_note: '',
    applied_at: 0,
    applied_by: 0,
    applied_note: '',
    created_at: periodStart + 60,
    updated_at: periodStart + 60,
  }

  const approvedDowngrade = {
    ...draftRecommendation,
    id: 72,
    supplier_scorecard_id: 42,
    score: 65,
    grade: 'C',
    recommended_action: 'downgrade',
    status: 'approved',
    reason: 'watch quality trend but keep supplier enabled',
    reviewed_at: periodStart + 300,
    reviewed_by: 1,
    review_note: 'approved supplier posture recommendation from dashboard',
  }

  const appliedDowngrade = {
    ...approvedDowngrade,
    status: 'applied',
    applied_at: periodStart + 600,
    applied_by: 1,
    applied_note: 'applied supplier posture recommendation from dashboard',
    supplier_status_before: 1,
    supplier_status_after: 1,
  }

  const activeRoutePreference = {
    id: 9001,
    supplier_id: supplier.id,
    source_posture_recommendation_id: approvedDowngrade.id,
    status: 'active',
    weight_percent: 25,
    reason: 'supplier_posture_recommendation #72 downgrade: grade=C score=65.000',
    effective_from: periodStart + 600,
    effective_to: 0,
    activated_at: periodStart + 600,
    activated_by: 1,
    disabled_at: 0,
    disabled_by: 0,
    operator_note: 'applied supplier posture recommendation from dashboard',
    created_at: periodStart + 600,
    updated_at: periodStart + 600,
  }

  let postureRecommendations = [draftRecommendation, approvedDowngrade]
  let routePreferences = []

  const ok = (data) => ({ success: true, message: '', data })
  const pageData = (items) =>
    ok({ page: 1, page_size: 100, total: items.length, items })
  const assert = (condition, message) => {
    if (!condition) throw new Error(message)
  }
  const waitForRequest = async (predicate, message) => {
    for (let i = 0; i < 50; i += 1) {
      if (requests.some(predicate)) return
      await page.waitForTimeout(100)
    }
    throw new Error(message)
  }
  const parseSearch = (search) => {
    const params = {}
    if (!search) return params
    for (const pair of search.split('&')) {
      if (!pair) continue
      const equalsIndex = pair.indexOf('=')
      const rawKey = equalsIndex === -1 ? pair : pair.slice(0, equalsIndex)
      const rawValue = equalsIndex === -1 ? '' : pair.slice(equalsIndex + 1)
      params[decodeURIComponent(rawKey)] = decodeURIComponent(rawValue)
    }
    return params
  }
  const parseRequestUrl = (rawUrl) => {
    const normalized = rawUrl.replace(/^https?:\/\/[^/]+/, '')
    const [pathname, search = ''] = normalized.split('?')
    return {
      pathname: pathname.replace(/\/+$/, ''),
      params: parseSearch(search),
      search,
    }
  }
  const fulfillJson = async (route, body) => {
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
    const path = url.pathname
    const method = request.method()
    const postData = request.postData()
    requests.push({
      method,
      path,
      search: url.search,
      params: url.params,
      postData,
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
    if (path === '/api/suppliers' && method === 'GET') {
      await fulfillJson(route, pageData([supplier]))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations' &&
      method === 'GET'
    ) {
      await fulfillJson(route, pageData(postureRecommendations))
      return
    }
    if (path === '/api/supplier_route_preferences' && method === 'GET') {
      await fulfillJson(route, pageData(routePreferences))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations/72/apply' &&
      method === 'POST'
    ) {
      postureRecommendations = [draftRecommendation, appliedDowngrade]
      routePreferences = [activeRoutePreference]
      await fulfillJson(route, ok(appliedDowngrade))
      return
    }
    if (
      path === '/api/reports/margin_summary' ||
      path === '/api/reports/quality_summary'
    ) {
      await fulfillJson(route, ok([]))
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
      [data-sonner-toaster] {
        display: none !important;
      }
    `,
  })
  await page.getByLabel('Period Start').fill('2026-06-23T05:03')
  await page.getByLabel('Period End').fill('2026-06-23T07:03')
  await page.getByRole('tab', { name: 'Posture' }).click()
  await page
    .getByText('Supplier Posture Recommendations', { exact: true })
    .waitFor()
  await page.getByText('No active supplier route preferences.').waitFor()

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Approved', exact: true })
    .click()
  await page
    .getByRole('row', { name: /#72/ })
    .getByRole('button', { name: 'Apply', exact: true })
    .click()
  await waitForRequest(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/72/apply',
    'apply POST was not observed after clicking Apply'
  )

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Applied', exact: true })
    .click()
  await page.getByText('Route Preference Active').waitFor()
  await page.getByText('Recommendation #72', { exact: true }).waitFor()
  await page.getByText('25%').first().waitFor()
  await page.getByText('supplier_posture_recommendation #72').waitFor()

  await page.evaluate(() => {
    for (const element of document.querySelectorAll('*')) {
      element.scrollLeft = 0
    }
  })
  await page
    .getByText('Active Route Preferences', { exact: true })
    .last()
    .evaluate((element) => {
      element.scrollIntoView({ block: 'center', inline: 'nearest' })
    })
  await page.waitForTimeout(300)
  await page.screenshot({ path: desktopScreenshot, fullPage: false })

  const routePreferenceGets = requests.filter(
    (entry) =>
      entry.method === 'GET' && entry.path === '/api/supplier_route_preferences'
  )
  const applyPosts = requests.filter(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/72/apply'
  )

  assert(applyPosts.length === 1, 'apply POST was not observed once')
  assert(
    routePreferenceGets.length >= 2,
    'route preference GET did not refetch after apply'
  )
  assert(
    routePreferenceGets.every((entry) => entry.params.status === 'active'),
    'route preference GET did not request active preferences'
  )
  assert(
    JSON.parse(applyPosts[0].postData || '{}').operator_note.includes(
      'applied supplier posture'
    ),
    'apply POST operator_note drifted'
  )

  await page.setViewportSize({ width: 390, height: 844 })
  await page
    .getByText('Active Route Preferences', { exact: true })
    .last()
    .scrollIntoViewIfNeeded()
  await page.waitForTimeout(300)
  await page.screenshot({ path: mobileScreenshot, fullPage: false })

  console.log(
    JSON.stringify({
      verified: true,
      screenshots: [desktopScreenshot, mobileScreenshot],
      routePreferenceGetCount: routePreferenceGets.length,
      applyPostCount: applyPosts.length,
    })
  )
}
