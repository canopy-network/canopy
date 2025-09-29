import React, { useState, useMemo } from 'react';
import TableCard from './Home/TableCard';
import TableFilters from './TableFilters';
import { useTableManifest, useTableFilters } from '../hooks/useTableManifest';

interface TableWithManifestProps {
    manifestKey: string;
    data: any[];
    loading?: boolean;
    currentPage?: number;
    totalCount?: number;
    onPageChange?: (page: number) => void;
    pageSize?: number;
    showFilters?: boolean;
    onFiltersChange?: (filters: Record<string, any>) => void;
}

const TableWithManifest: React.FC<TableWithManifestProps> = ({
    manifestKey,
    data,
    loading = false,
    currentPage = 1,
    totalCount = 0,
    onPageChange,
    pageSize = 10,
    showFilters = true,
    onFiltersChange
}) => {
    const [filters, setFilters] = useState<Record<string, any>>({});
    
    const manifest = useTableManifest(manifestKey);
    const filterManifest = useTableFilters(manifestKey);

    // Apply filters to data
    const filteredData = useMemo(() => {
        if (!filterManifest?.enabled || Object.keys(filters).length === 0) {
            return data;
        }

        return data.filter(item => {
            return filterManifest.fields.every(field => {
                const filterValue = filters[field.key];
                if (!filterValue || filterValue === 'all' || filterValue === '') {
                    return true;
                }

                const itemValue = item[field.key];

                switch (field.type) {
                    case 'select':
                        return itemValue === filterValue;
                    case 'text':
                        return itemValue?.toString().toLowerCase().includes(filterValue.toLowerCase());
                    case 'range': {
                        const numValue = parseFloat(itemValue);
                        const filterNum = parseFloat(filterValue);
                        return !isNaN(numValue) && numValue >= filterNum;
                    }
                    default:
                        return true;
                }
            });
        });
    }, [data, filters, filterManifest]);

    const handleFiltersChange = (newFilters: Record<string, any>) => {
        setFilters(newFilters);
        onFiltersChange?.(newFilters);
    };

    const handleApplyFilters = (newFilters: Record<string, any>) => {
        setFilters(newFilters);
        onFiltersChange?.(newFilters);
    };

    const handleResetFilters = () => {
        setFilters({});
        onFiltersChange?.({});
    };

    if (!manifest) {
        return <div>Table manifest not found: {manifestKey}</div>;
    }

    return (
        <div>
            {showFilters && filterManifest && (
                <TableFilters
                    manifest={filterManifest}
                    onFiltersChange={handleFiltersChange}
                    onApplyFilters={handleApplyFilters}
                    onResetFilters={handleResetFilters}
                />
            )}
            
            <TableCard
                title={manifest.title}
                manifestKey={manifestKey}
                data={filteredData}
                loading={loading}
                paginate={manifest.pagination?.enabled || false}
                currentPage={currentPage}
                totalCount={totalCount}
                onPageChange={onPageChange}
                pageSize={pageSize}
                columns={[]}
                rows={[]}
            />
        </div>
    );
};

export default TableWithManifest;
