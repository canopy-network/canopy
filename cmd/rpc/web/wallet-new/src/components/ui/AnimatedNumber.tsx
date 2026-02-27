import React from 'react'
import NumberFlow from '@number-flow/react'

interface AnimatedNumberProps {
  value: number
  format?: {
    notation?: 'standard' | 'compact'
    maximumFractionDigits?: number
    minimumFractionDigits?: number
  }
  locales?: Intl.LocalesArgument
  prefix?: string
  suffix?: string
  className?: string
  trend?: number | ((oldValue: number, value: number) => number)
  animated?: boolean
  respectMotionPreference?: boolean
}

const AnimatedNumber: React.FC<AnimatedNumberProps> = ({
  value,
  format,
  locales = 'en-US',
  prefix,
  suffix,
  className = '',
  trend,
  animated = true,
  respectMotionPreference = true,
}) => {
  // Ensure value is a valid number
  const numericValue = typeof value === 'number' && !isNaN(value) ? value : 0;

  return (
    <NumberFlow
      value={numericValue}
      format={format}
      locales={locales}
      prefix={prefix}
      suffix={suffix}
      className={className}
      trend={trend}
      animated={animated}
      respectMotionPreference={respectMotionPreference}
      transformTiming={{ duration: 600, easing: 'ease-out' }}
      spinTiming={{ duration: 600, easing: 'ease-out' }}
      opacityTiming={{ duration: 300, easing: 'ease-out' }}
    />
  )
}

export default AnimatedNumber
