import React from 'react'

interface PageHeaderProps {
  title: React.ReactNode
  subtitle: React.ReactNode
  actions?: React.ReactNode
  className?: string
}

export const PageHeader = ({ title, subtitle, actions, className = '' }: PageHeaderProps) => {
  return (
    <div className={`flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between ${className}`.trim()}>
      <div className="min-w-0">
        <h1 className="wallet-page-title">{title}</h1>
        <p className="wallet-page-subtitle">{subtitle}</p>
      </div>
      {actions ? <div className="lg:self-start">{actions}</div> : null}
    </div>
  )
}

export default PageHeader
