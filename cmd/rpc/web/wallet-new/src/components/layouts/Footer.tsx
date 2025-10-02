import React from 'react';
import { motion } from 'framer-motion';

export const Footer = (): JSX.Element => {
    const containerVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: {
                duration: 0.6,
                staggerChildren: 0.1
            }
        }
    };

    const itemVariants = {
        hidden: { opacity: 0, y: 10 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const linkVariants = {
        hover: {
            scale: 1.05,
            color: "#6fe3b4",
            transition: { duration: 0.2 }
        }
    };

    return (
        <motion.footer
            className="bg-secondary border-t border-bg-accent mt-auto"
            initial="hidden"
            animate="visible"
            variants={containerVariants}
        >
            <div className="px-6 py-6">
                <motion.div
                    className="flex flex-wrap justify-center items-center gap-8"
                    variants={containerVariants}
                >
                    <motion.a
                        href="#"
                        className="text-gray-300 hover:text-[#6fe3b4] transition-colors duration-200 text-sm font-medium"
                        variants={itemVariants}
                        whileHover="hover"
                        animate="visible"
                        custom={0}
                    >
                        Terms of Service
                    </motion.a>

                    <motion.a
                        href="#"
                        className="text-gray-300 hover:text-[#6fe3b4] transition-colors duration-200 text-sm font-medium"
                        variants={itemVariants}
                        whileHover="hover"
                        animate="visible"
                        custom={1}
                    >
                        Privacy Policy
                    </motion.a>

                    <motion.a
                        href="#"
                        className="text-gray-300 hover:text-[#6fe3b4] transition-colors duration-200 text-sm font-medium"
                        variants={itemVariants}
                        whileHover="hover"
                        animate="visible"
                        custom={2}
                    >
                        Security Guide
                    </motion.a>

                    <motion.a
                        href="#"
                        className="text-gray-300 hover:text-[#6fe3b4] transition-colors duration-200 text-sm font-medium"
                        variants={itemVariants}
                        whileHover="hover"
                        animate="visible"
                        custom={3}
                    >
                        Support
                    </motion.a>
                </motion.div>
            </div>
        </motion.footer>
    );
};
