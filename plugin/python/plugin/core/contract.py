"""
Contract implementation for Canopy blockchain plugin.

Handles send transaction validation and execution with state management.
"""

import random
from typing import Optional, Dict, Any, Union, Protocol, cast
from dataclasses import dataclass
from ..config import Config
from ..proto_utils import marshal, unmarshal
from ..proto import (
    PluginCheckRequest,
    PluginCheckResponse,
    PluginDeliverRequest,
    PluginDeliverResponse,
    PluginGenesisRequest,
    PluginGenesisResponse,
    PluginBeginRequest,
    PluginBeginResponse,
    PluginEndRequest,
    PluginEndResponse,
    MessageSend,
    PluginKeyRead,
    PluginStateReadRequest,
    PluginStateReadResponse,
    PluginStateWriteRequest,
    PluginStateWriteResponse,
    PluginSetOp,
    PluginDeleteOp,
    PluginFSMConfig,
    FeeParams,
    Pool,
)
from .keys import (
    key_for_account,
    key_for_fee_pool,
    key_for_fee_params,
)
from .validation import (
    validate_address,
    validate_amount,
    normalize_address,
    normalize_amount,
)
from ..proto import Account, Pool, FeeParams
from .exceptions import (
    # Exception classes
    PluginException,
    InvalidAddressError,
    InvalidAmountError,
    InsufficientFundsError,
    FeeBelowLimitError,
    UnsupportedMessageTypeError,
    PluginNotInitializedError,
    ParameterError,
    # Response helper functions
    create_check_error_response_from_exception,
    create_deliver_error_response_from_exception,
)


class SocketClientPlugin(Protocol):
    """Protocol for socket client plugin interface."""

    async def state_read(
        self, contract: "Contract", request: PluginStateReadRequest
    ) -> PluginStateReadResponse: ...

    async def state_write(
        self, contract: "Contract", request: PluginStateWriteRequest
    ) -> PluginStateWriteResponse: ...


@dataclass
class ContractOptions:
    """Options for creating a Contract instance."""

    config: Optional[Config] = None
    fsm_config: Optional[PluginFSMConfig] = None
    plugin: Optional[SocketClientPlugin] = None
    fsm_id: Optional[int] = None


# Plugin configuration
CONTRACT_CONFIG = {
    "name": "send",
    "id": 1,
    "version": 1,
    "supported_transactions": ["send"],
}


class Contract:
    """
    Contract class for blockchain transaction validation and execution.
    """

    def __init__(self, options: Optional[ContractOptions] = None):
        """Initialize contract with optional configuration."""
        if options is None:
            options = ContractOptions()

        self.config = options.config
        self.fsm_config = options.fsm_config
        self.plugin = options.plugin
        self.fsm_id = options.fsm_id

    def genesis(self, _request: PluginGenesisRequest) -> PluginGenesisResponse:
        """Genesis block processing implementation."""
        return PluginGenesisResponse()

    def begin_block(self, _request: PluginBeginRequest) -> PluginBeginResponse:
        """Begin block processing implementation."""
        return PluginBeginResponse()

    async def check_tx(
        self, id: int, request: PluginCheckRequest
    ) -> PluginCheckResponse:
        """
        CheckTx - validate transaction without state changes.

        Args:
            request: Transaction validation request

        Returns:
            Validation result with authorized signers
        """
        try:
            if not self.plugin or not self.config:
                raise PluginNotInitializedError()

            # Validate fee against state parameters
            fee_params_response = await self.plugin.state_read(
                self,
                PluginStateReadRequest(
                    keys=[
                        PluginKeyRead(
                            query_id=id,
                            key=key_for_fee_params(),
                        )
                    ]
                ),
            )

            if fee_params_response.HasField("error"):
                response = PluginCheckResponse()
                response.error.CopyFrom(fee_params_response.error)
                return response

            # Convert bytes into fee parameters
            if (
                not fee_params_response.results
                or not fee_params_response.results[0].entries
            ):
                raise ParameterError("Fee parameters not found")

            fee_params_bytes = fee_params_response.results[0].entries[0].value
            if not fee_params_bytes:
                raise ParameterError("Fee parameters not found")

            min_fees = unmarshal(FeeParams, fee_params_bytes)
            if not min_fees:
                raise ParameterError("Failed to decode fee parameters")

            # Check for minimum fee
            try:
                request_fee = normalize_amount(request.tx.fee)
                min_send_fee = normalize_amount(min_fees.send_fee)
            except Exception as error:
                raise ParameterError(f"Failed to normalize fee: {error}")

            if request_fee < min_send_fee:
                raise FeeBelowLimitError(fee=request_fee, minimum=min_send_fee)

            if request.tx.msg.type_url.endswith("/types.MessageSend"):
                message_send = MessageSend()
                message_send.ParseFromString(request.tx.msg.value)

                try:
                    return self._check_message_send(message_send)
                except PluginException as e:
                    return create_check_error_response_from_exception(e)
            else:
                raise UnsupportedMessageTypeError(request.tx.msg.type_url)

        except PluginException as e:
            return create_check_error_response_from_exception(e)
        except Exception as err:
            return create_check_error_response_from_exception(
                PluginException(str(err), code=1, module="contract")
            )

    async def deliver_tx(self, request: PluginDeliverRequest) -> PluginDeliverResponse:
        """
        DeliverTx - execute transaction with state changes.

        Args:
            request: Transaction execution request

        Returns:
            Execution result
        """
        try:
            if request.tx.msg.type_url.endswith("/types.MessageSend"):
                message_send = MessageSend()
                message_send.ParseFromString(request.tx.msg.value)

                try:
                    return await self._deliver_message_send(
                        message_send, request.tx.fee
                    )
                except PluginException as e:
                    return create_deliver_error_response_from_exception(e)
            else:
                raise UnsupportedMessageTypeError(request.tx.msg.type_url)

        except PluginException as e:
            return create_deliver_error_response_from_exception(e)
        except Exception as err:
            return create_deliver_error_response_from_exception(
                PluginException(str(err), code=1, module="contract")
            )

    def end_block(self, _request: PluginEndRequest) -> PluginEndResponse:
        """End block processing implementation."""
        return PluginEndResponse()

    def _check_message_send(self, msg: MessageSend) -> PluginCheckResponse:
        """
        Validate MessageSend without state changes.

        Args:
            msg: MessageSend to validate

        Returns:
            Validation result with authorized signers

        Raises:
            InvalidAddressError: If from_address or to_address is invalid
            InvalidAmountError: If amount is invalid
        """
        # Check sender address (must be exactly 20 bytes)
        if not validate_address(msg.from_address):
            raise InvalidAddressError(msg.from_address)

        # Check recipient address (must be exactly 20 bytes)
        if not validate_address(msg.to_address):
            raise InvalidAddressError(msg.to_address)

        # Check amount (must be greater than 0)
        if not validate_amount(msg.amount):
            raise InvalidAmountError(msg.amount)

        # Return authorized signers (sender must sign)
        response = PluginCheckResponse()
        response.recipient = msg.to_address
        response.authorized_signers.append(msg.from_address)
        return response

    def _generate_query_ids(self) -> Dict[str, int]:
        """
        Generate random query IDs for batch state operations.

        Returns:
            Dict containing fromQueryId, toQueryId, and feeQueryId
        """
        return {
            "from_query_id": random.randint(0, 2**53 - 1),
            "to_query_id": random.randint(0, 2**53 - 1),
            "fee_query_id": random.randint(0, 2**53 - 1),
        }

    async def _read_deliver_message_required_data(
        self, msg: MessageSend
    ) -> Dict[str, Optional[bytes]]:
        """
        Read accounts and fee pool from state.

        Args:
            msg: MessageSend containing addresses

        Returns:
            Dict with account and fee pool bytes
        """
        if not self.plugin or not self.config:
            raise PluginNotInitializedError()

        query_ids = self._generate_query_ids()

        # Calculate state keys
        from_key = key_for_account(msg.from_address)
        to_key = key_for_account(msg.to_address)
        fee_pool_key = key_for_fee_pool(self.config.chain_id)

        # Batch read accounts and fee pool from state
        response = await self.plugin.state_read(
            self,
            PluginStateReadRequest(
                keys=[
                    PluginKeyRead(query_id=query_ids["fee_query_id"], key=fee_pool_key),
                    PluginKeyRead(query_id=query_ids["from_query_id"], key=from_key),
                    PluginKeyRead(query_id=query_ids["to_query_id"], key=to_key),
                ]
            ),
        )

        if response.HasField("error"):
            raise PluginException(
                f"State read error: {response.error.msg}", code=4, module="plugin"
            )

        # Parse response results by query ID
        from_bytes = None
        to_bytes = None
        fee_pool_bytes = None

        for result in response.results:
            if result.query_id == query_ids["from_query_id"]:
                from_bytes = result.entries[0].value if result.entries else None
            elif result.query_id == query_ids["to_query_id"]:
                to_bytes = result.entries[0].value if result.entries else None
            elif result.query_id == query_ids["fee_query_id"]:
                fee_pool_bytes = result.entries[0].value if result.entries else None

        return {
            "from_bytes": from_bytes,
            "to_bytes": to_bytes,
            "fee_pool_bytes": fee_pool_bytes,
        }

    def _unmarshal_deliver_message_required_data(
        self,
        from_bytes: Optional[bytes],
        to_bytes: Optional[bytes],
        fee_pool_bytes: Optional[bytes],
        msg: MessageSend,
    ) -> Dict[str, Any]:
        """
        Unmarshal account and pool data from state bytes.

        Args:
            from_bytes: From account bytes
            to_bytes: To account bytes
            fee_pool_bytes: Fee pool bytes
            msg: MessageSend containing addresses

        Returns:
            Dict with unmarshaled accounts, pool, and normalized fromAmount
        """
        from_account = unmarshal(Account, from_bytes) if from_bytes else None

        # Extract to address from message
        to_addr = msg.to_address

        try:
            to_account = unmarshal(Account, to_bytes) if to_bytes else None
            if not to_account:
                to_account = Account(address=to_addr, amount=0)
        except Exception:
            to_account = Account(address=to_addr, amount=0)

        try:
            fee_pool = unmarshal(Pool, fee_pool_bytes) if fee_pool_bytes else None
            if not fee_pool:
                fee_pool = Pool(amount=0)
        except Exception:
            fee_pool = Pool(amount=0)

        from_amount = from_account.amount if from_account else 0

        return {
            "from_account": from_account,
            "to_account": to_account,
            "fee_pool": fee_pool,
            "from_amount": from_amount,
        }

    async def _deliver_message_send(
        self, msg: MessageSend, fee: Union[int, str]
    ) -> PluginDeliverResponse:
        """
        Process MessageSend with state changes.

        Args:
            msg: MessageSend to execute
            fee: Transaction fee

        Returns:
            Execution result
        """
        try:
            transaction_fee = normalize_amount(fee)

            if not self.plugin or not self.config:
                raise PluginNotInitializedError()

            # Read accounts and fee pool from state
            state_data = await self._read_deliver_message_required_data(msg)

            # Unmarshal account and pool data
            unmarshaled_data = self._unmarshal_deliver_message_required_data(
                state_data["from_bytes"],
                state_data["to_bytes"],
                state_data["fee_pool_bytes"],
                msg,
            )

            from_account = unmarshaled_data["from_account"]
            to_account = unmarshaled_data["to_account"]
            fee_pool = unmarshaled_data["fee_pool"]
            from_amount = unmarshaled_data["from_amount"]

            # Extract amount from message
            message_amount = msg.amount

            message_amount = normalize_amount(message_amount)
            amount_to_deduct = message_amount + transaction_fee

            # Check sufficient funds
            if from_amount < amount_to_deduct:
                raise InsufficientFundsError(
                    required=amount_to_deduct, available=from_amount
                )

            # Calculate state keys
            from_addr = msg.from_address
            to_addr = msg.to_address

            from_key = key_for_account(from_addr)
            to_key = key_for_account(to_addr)
            fee_pool_key = key_for_fee_pool(self.config.chain_id)

            # Calculate updated balances
            updated_from_account = Account(
                address=normalize_address(from_addr),
                amount=from_amount - amount_to_deduct,
            )

            # Handle self-transfer optimization
            is_self_transfer = from_key == to_key

            if is_self_transfer:
                updated_to_account = Account(
                    address=normalize_address(to_addr),
                    amount=from_amount
                    - transaction_fee,  # Only deduct fee for self-transfer
                )
            else:
                updated_to_account = Account(
                    address=normalize_address(to_addr),
                    amount=to_account.amount + message_amount,
                )

            # Update fee pool
            updated_fee_pool = Pool(
                id=self.config.chain_id, amount=fee_pool.amount + transaction_fee
            )

            # Prepare state write operations
            sets = [PluginSetOp(key=fee_pool_key, value=marshal(updated_fee_pool))]
            deletes = []

            # Handle account deletion when balance reaches zero
            if updated_from_account.amount == 0 and not is_self_transfer:
                deletes.append(PluginDeleteOp(key=from_key))
            else:
                sets.append(
                    PluginSetOp(key=from_key, value=marshal(updated_from_account))
                )

            if not is_self_transfer:
                sets.append(PluginSetOp(key=to_key, value=marshal(updated_to_account)))

            # Execute batch state write
            write_response = await self.plugin.state_write(
                self, PluginStateWriteRequest(sets=sets, deletes=deletes)
            )

            # Check if protobuf response has error field
            response = PluginDeliverResponse()
            if write_response.HasField("error"):
                response.error.CopyFrom(write_response.error)
            return response

        except PluginException as e:
            return create_deliver_error_response_from_exception(e)
        except Exception as err:
            return create_deliver_error_response_from_exception(
                PluginException(str(err), code=1, module="contract")
            )
