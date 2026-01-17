import { Footer, Layout, Navbar } from 'nextra-theme-docs'
import { Banner, Head } from 'nextra/components'
import { getPageMap } from 'nextra/page-map'
import 'nextra-theme-docs/style.css'
import './globals.css'
import { readFileSync } from 'fs'
import { join } from 'path'

// Get basePath from environment
const basePath = process.env.NODE_ENV === 'production' ? '/mind-palace' : ''

// Read version from VERSION file at build time
let version = 'unknown'
try {
  version = readFileSync(join(process.cwd(), '..', '..', '..', 'VERSION'), 'utf8').trim()
} catch {
  try {
    version = readFileSync(join(process.cwd(), '..', 'VERSION'), 'utf8').trim()
  } catch {
    // Fallback to package.json
    try {
      const pkg = JSON.parse(readFileSync(join(process.cwd(), 'package.json'), 'utf8'))
      version = pkg.version
    } catch {
      version = '0.0.0'
    }
  }
}

export const metadata = {
  title: {
    default: 'Mind Palace',
    template: '%s - Mind Palace'
  },
  description: 'A deterministic context system for codebases',
  openGraph: {
    title: 'Mind Palace Documentation',
    description: 'A deterministic context system for codebases',
  },
  icons: {
    icon: `${basePath}/favicon.png`,
  },
}

const banner = (
  <Banner storageKey={`mind-palace-${version}`}>
    <a href="https://github.com/koksalmehmet/mind-palace/releases" target="_blank">
      Mind Palace {version} is out. Check it out →
    </a>
  </Banner>
)

const Logo = () => (
  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
    <svg viewBox="0 0 512 512" width="28" height="28">
      <g fill="#6B5B95">
        <path d="M499.51,335.772l-46.048-130.234C445.439,90.702,349.802,0,232.917,0C110.768,0,11.759,99.019,11.759,221.158
          v69.684C11.759,412.982,110.768,512,232.917,512c100.571,0,185.406-67.154,212.256-159.054h42.186c4.181,0,8.104-2.032,10.518-5.45
          C500.291,344.088,500.895,339.712,499.51,335.772z M328.82,214.59c2.511,14.166-2.495,37.128-47.051,37.128
          c-21.355,0-51.382,0-61.731,0c0,33.737-50.903,25.819-68.779,25.819c-17.911,0-20.832-19.19-17.565-24.178
          c-55.846-0.417-63.701-58.749-49.988-84.196C89.573,99.96,159.585,50.77,229.242,50.77c91.661,0,85.59,30.009,95.805,30.861
          c25.03,2.023,59.219,31.269,59.219,65.415C384.267,181.244,370.074,214.59,328.82,214.59z"/>
      </g>
      <g fill="#FFFFFF" opacity="0.95">
        <path d="M259.252,135.818c0,3.453,2.76,6.258,6.204,6.258h9.799v5.184c0,3.453,2.84,6.284,6.213,6.284
          c3.524,0,6.32-2.831,6.32-6.284v-24.374c0-5.299,0.923-8.99,2.61-10.642c1.89-1.988,5.92-1.988,10.154-1.837h25.97
          c3.533,0,6.249-2.831,6.249-6.302c0-3.39-2.716-6.23-6.249-6.23h-25.935c-5.964-0.054-13.456-0.106-19.012,5.45
          c-4.26,4.207-6.319,10.571-6.319,19.562v6.622h-9.799C262.012,129.507,259.252,132.365,259.252,135.818z"/>
        <path d="M271.233,175.341c-1.082,0-1.98,0-2.866,0h-50.45c-7.908,0-13.65-1.73-16.908-5.192
          c-5.317-5.618-4.891-15.541-4.501-23.6c0.08-1.935,0.16-3.745,0.16-5.335c0-3.487-2.751-6.256-6.159-6.256
          c-3.533,0-6.329,2.769-6.329,6.256c0,1.394-0.08,3.054-0.16,4.82c-0.452,9.249-1.082,23.237,7.838,32.681
          c5.805,6.088,14.538,9.142,26.059,9.142h50.45c0.923,0,1.97,0,3.071,0c8.521-0.125,21.462-0.311,28.34,6.426
          c3.596,3.515,5.3,8.654,5.3,15.737c0,3.399,2.84,6.257,6.249,6.257c3.506,0,6.248-2.858,6.248-6.257
          c0-10.429-3.036-18.719-9-24.604C297.932,174.924,281.192,175.19,271.233,175.341z"/>
      </g>
    </svg>
    <span style={{ fontWeight: 700, fontSize: '1.1rem' }}>Mind Palace</span>
  </div>
)

const navbar = (
  <Navbar
    logo={<Logo />}
    projectLink="https://github.com/koksalmehmet/mind-palace"
  />
)

const footer = (
  <Footer>
    {new Date().getFullYear()} © Mind Palace. Licensed under MIT License
  </Footer>
)

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" dir="ltr" suppressHydrationWarning>
      <Head>
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <link rel="icon" type="image/png" href={`${basePath}/favicon.png`} />
      </Head>
      <body>
        <Layout
          banner={banner}
          navbar={navbar}
          pageMap={await getPageMap()}
          docsRepositoryBase="https://github.com/koksalmehmet/mind-palace/tree/main/apps/docs/content"
          footer={footer}
          editLink="Edit this page on GitHub →"
          feedback={{ content: 'Question? Give us feedback →', labels: 'feedback' }}
          sidebar={{ defaultMenuCollapseLevel: 1, toggleButton: true }}
          toc={{ backToTop: true }}
        >
          {children}
        </Layout>
      </body>
    </html>
  )
}

