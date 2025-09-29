import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import AnimatedNumber from '../components/AnimatedNumber';
import tableManifests from '../data/table-manifests.json';

export interface TableManifest {
  title: string;
  columns: ColumnManifest[];
  filters?: FilterManifest;
  pagination?: PaginationManifest;
}

export interface ColumnManifest {
  key: string;
  label: string;
  type: 'text' | 'number' | 'address' | 'hash' | 'datetime' | 'relative-time';
  format?: 'truncate' | 'badge' | 'currency' | 'percentage' | 'duration' | 'animated';
  truncate?: number;
  decimals?: number;
  suffix?: string;
  link?: string;
  icon?: string;
  color?: string;
  colorMapping?: Record<string, { color: string }>;
}

export interface FilterManifest {
  enabled: boolean;
  fields: FilterFieldManifest[];
}

export interface FilterFieldManifest {
  key: string;
  type: 'select' | 'text' | 'range';
  label: string;
  placeholder?: string;
  options?: Array<{ value: string; label: string }>;
  min?: number;
  max?: number;
  step?: number;
  suffix?: string;
}

export interface PaginationManifest {
  enabled: boolean;
  pageSize: number;
  showTotal: boolean;
}

export const useTableManifest = (tableKey: string): TableManifest | null => {
  return useMemo(() => {
    const manifest = tableManifests.tables[tableKey as keyof typeof tableManifests.tables];
    return manifest || null;
  }, [tableKey]);
};

export const useTableColumns = (tableKey: string) => {
  const manifest = useTableManifest(tableKey);
  
  return useMemo(() => {
    if (!manifest) return [];
    
    return manifest.columns.map(col => ({
      label: col.label,
      key: col.key
    }));
  }, [manifest]);
};

export const useTableFilters = (tableKey: string) => {
  const manifest = useTableManifest(tableKey);
  
  return useMemo(() => {
    if (!manifest?.filters?.enabled) return null;
    return manifest.filters;
  }, [manifest]);
};

export const useTablePagination = (tableKey: string) => {
  const manifest = useTableManifest(tableKey);
  
  return useMemo(() => {
    if (!manifest?.pagination?.enabled) return null;
    return manifest.pagination;
  }, [manifest]);
};

// Helper function to format cell values based on manifest
export const formatCellValue = (value: any, column: ColumnManifest, data?: any): React.ReactNode => {
  if (value === null || value === undefined) return 'N/A';

  switch (column.type) {
    case 'number':
      return formatNumberValue(value, column);
    case 'address':
    case 'hash':
      return formatAddressValue(value, column, data);
    case 'datetime':
      return formatDateTimeValue(value, column);
    case 'relative-time':
      return formatRelativeTimeValue(value, column);
    case 'text':
    default:
      return formatTextValue(value, column);
  }
};

const formatNumberValue = (value: number, column: ColumnManifest): React.ReactNode => {
  const numValue = typeof value === 'string' ? parseFloat(value) : value;
  
  if (isNaN(numValue)) return 'N/A';

  let formattedValue: React.ReactNode = numValue;

  if (column.format === 'animated') {
    formattedValue = (
      <AnimatedNumber 
        value={numValue} 
        className={column.color ? `text-${column.color}` : 'text-gray-300'}
        format={{ 
          maximumFractionDigits: column.decimals || 0,
          minimumFractionDigits: column.decimals || 0
        }}
      />
    );
  } else if (column.format === 'currency') {
    formattedValue = (
      <>
        <AnimatedNumber 
          value={numValue} 
          format={{ 
            maximumFractionDigits: column.decimals || 2,
            minimumFractionDigits: column.decimals || 2
          }}
        />
        {column.suffix && <span className="ml-1">{column.suffix}</span>}
      </>
    );
  } else if (column.format === 'percentage') {
    formattedValue = (
      <>
        <AnimatedNumber 
          value={numValue} 
          format={{ 
            maximumFractionDigits: column.decimals || 2,
            minimumFractionDigits: column.decimals || 2
          }}
        />
        {column.suffix && <span className="ml-1">{column.suffix}</span>}
      </>
    );
  } else if (column.format === 'duration') {
    formattedValue = (
      <>
        <AnimatedNumber 
          value={numValue} 
          format={{ 
            maximumFractionDigits: column.decimals || 2,
            minimumFractionDigits: column.decimals || 2
          }}
        />
        {column.suffix && <span className="ml-1">{column.suffix}</span>}
      </>
    );
  } else if (column.format === 'badge') {
    const colorClass = column.color ? `text-${column.color}` : 'text-gray-300';
    formattedValue = (
      <span className={`px-2 py-1 rounded-full text-xs font-medium bg-${column.color}/20 ${colorClass}`}>
        <AnimatedNumber value={numValue} />
      </span>
    );
  }

  return formattedValue;
};

const formatAddressValue = (value: string, column: ColumnManifest, data?: any): React.ReactNode => {
  let displayValue = value;
  
  if (column.format === 'truncate' && column.truncate && value.length > column.truncate) {
    displayValue = `${value.slice(0, column.truncate)}...${value.slice(-4)}`;
  }

  const content = (
    <span className={`font-mono text-sm ${column.color ? `text-${column.color}` : 'text-gray-300'}`}>
      {displayValue}
    </span>
  );

  if (column.link && data) {
    const linkPath = column.link.replace('{{value}}', value).replace('{{address}}', data.address || value);
    return (
      <Link to={linkPath} className="hover:underline">
        {content}
      </Link>
    );
  }

  return content;
};

const formatDateTimeValue = (value: string, column: ColumnManifest): React.ReactNode => {
  try {
    const date = new Date(value);
    if (isNaN(date.getTime())) return 'N/A';
    
    return (
      <span className="text-gray-300 font-mono text-sm">
        {date.toLocaleString()}
      </span>
    );
  } catch {
    return 'N/A';
  }
};

const formatRelativeTimeValue = (value: string, column: ColumnManifest): React.ReactNode => {
  try {
    const date = new Date(value);
    if (isNaN(date.getTime())) return 'N/A';
    
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    let relativeTime = '';
    if (diffMins < 1) relativeTime = 'Just now';
    else if (diffMins < 60) relativeTime = `${diffMins}m ago`;
    else if (diffHours < 24) relativeTime = `${diffHours}h ago`;
    else relativeTime = `${diffDays}d ago`;

    return (
      <span className="text-gray-400 text-sm">
        {relativeTime}
      </span>
    );
  } catch {
    return 'N/A';
  }
};

const formatTextValue = (value: string, column: ColumnManifest): React.ReactNode => {
  let displayValue = value;
  
  if (column.format === 'truncate' && column.truncate && value.length > column.truncate) {
    displayValue = `${value.slice(0, column.truncate)}...`;
  }

  if (column.format === 'badge' && column.colorMapping) {
    const colorConfig = column.colorMapping[value];
    const colorClass = colorConfig ? `text-${colorConfig.color}-400 bg-${colorConfig.color}-500/20` : 'text-gray-400 bg-gray-500/20';
    
    return (
      <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${colorClass}`}>
        {displayValue}
      </span>
    );
  }

  return (
    <span className="text-gray-300 text-sm">
      {displayValue}
    </span>
  );
};
